package controllers

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	crypt_rand "crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/rand"
	"strings"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Generate a random string with length pwdSize
func createPwd(pwdSize int) string {
	const chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	pwd := make([]byte, pwdSize)
	for i := range pwd {
		pwd[i] = chars[rand.Intn(len(chars))]
	}
	return string(pwd)
}

// Retrieve specific keys from secret object
func (r *PulpReconciler) retrieveSecretData(ctx context.Context, secretName, secretNamespace string, keys ...string) (map[string]string, error) {
	found := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: secretNamespace}, found)
	if err != nil {
		return nil, err
	}

	secret := map[string]string{}
	for _, key := range keys {
		secret[key] = string(found.Data[key])
	}

	return secret, nil
}

// Get signing key fingerprint from secret object
func (r *PulpReconciler) getSigningKeyFingerprint(secretName, secretNamespace string) (string, error) {

	ctx := context.TODO()
	secretData, err := r.retrieveSecretData(ctx, secretName, secretNamespace, "signing_service.gpg")
	if err != nil {
		return "", err
	}

	// "convert" to Reader to be used by ReadArmoredKeyRing
	secretReader := strings.NewReader(secretData["signing_service.gpg"])

	// Read public key
	keyring, err := openpgp.ReadArmoredKeyRing(secretReader)
	if err != nil {
		fmt.Println("Read Key Ring Error! " + err.Error())
		return "", err
	}

	fingerPrint := keyring[0].PrimaryKey.Fingerprint
	return strings.ToUpper(hex.EncodeToString(fingerPrint[:])), nil

}

func convertRawPulpSettings(pulp *repomanagerv1alpha1.Pulp) string {
	raw_settings := pulp.Spec.PulpSettings.RawSettings.Raw
	var settingsJson map[string]interface{}
	json.Unmarshal(raw_settings, &settingsJson)

	var convertedSettings string
	for k, v := range settingsJson {
		switch v.(type) {
		case map[string]interface{}:
			rawMapping, _ := json.Marshal(v)
			convertedSettings = convertedSettings + fmt.Sprintln(strings.ToUpper(k), "=", strings.Replace(string(rawMapping), "\"", "'", -1))
		default:
			convertedSettings = convertedSettings + fmt.Sprintf("%v = \"%v\"\n", strings.ToUpper(k), v)
		}
	}

	return convertedSettings
}

func genTokenAuthKey() (string, string) {
	newKey, _ := ecdsa.GenerateKey(elliptic.P256(), crypt_rand.Reader)
	pubKeyDER, _ := x509.MarshalPKIXPublicKey(&newKey.PublicKey)
	ecDER, _ := x509.MarshalECPrivateKey(newKey)

	privateKey := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: ecDER}))
	publicKey := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyDER}))

	return privateKey, publicKey
}

func (r *PulpReconciler) updateStatus(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, conditionStatus metav1.ConditionStatus, conditionType, conditionReason, conditionMessage string) {

	// if we are updating a status it means that operator didn't finish its execution

	if !v1.IsStatusConditionPresentAndEqual(pulp.Status.Conditions, cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType)+"-Operator-Finished-Execution", metav1.ConditionTrue) {
		v1.SetStatusCondition(&pulp.Status.Conditions, metav1.Condition{
			Type:               cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Operator-Finished-Execution",
			Status:             metav1.ConditionFalse,
			Reason:             "OperatorRunning",
			LastTransitionTime: metav1.Now(),
			Message:            pulp.Name + " operator tasks running",
		})
	}

	v1.SetStatusCondition(&pulp.Status.Conditions, metav1.Condition{
		Type:               cases.Title(language.English, cases.Compact).String(conditionType),
		Status:             conditionStatus,
		Reason:             conditionReason,
		LastTransitionTime: metav1.Now(),
		Message:            conditionMessage,
	})
	r.Status().Update(ctx, pulp)
}

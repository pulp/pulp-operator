package controllers

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"

	"golang.org/x/crypto/openpgp"
	corev1 "k8s.io/api/core/v1"
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

	ctx := context.Background()
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

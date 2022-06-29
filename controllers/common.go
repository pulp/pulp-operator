package controllers

import (
	"context"
	"math/rand"

	"github.com/go-logr/logr"
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

// WIP - shoud accept a variadic number of keys to avoid making a lot of api calls to retrieve each key
// Retrieve an specific key from secret object
func (r *PulpReconciler) retrieveSecretData(ctx context.Context, key, secretName, secretNamespace string, log logr.Logger) ([]byte, error) {
	found := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: secretNamespace}, found)
	if err != nil {
		log.Error(err, "Secret Not Found!", "Secret.Namespace", secretNamespace, "Secret.Name", secretName)
		return nil, err
	}

	return found.Data[key], nil
}

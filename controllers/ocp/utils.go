/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ocp

import (
	"context"

	"github.com/go-logr/logr"
	configv1 "github.com/openshift/api/config/v1"
	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	rhOperatorPullSecretName = "redhat-operators-pull-secret"
	caConfigMapName          = "user-ca-bundle"
)

// CreateRHOperatorPullSecret creates a default secret called redhat-operators-pull-secret
func CreateRHOperatorPullSecret(r client.Client, ctx context.Context, namespace string) error {
	log := logr.Logger{}

	// Get redhat-operators-pull-secret
	defaultSecret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: rhOperatorPullSecretName, Namespace: namespace}, defaultSecret)

	// Create the secret in case it is not found
	if err != nil && k8s_errors.IsNotFound(err) {
		defaultSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      rhOperatorPullSecretName,
				Namespace: namespace,
			},
			StringData: map[string]string{
				"operator": "pulp",
			},
		}
		r.Create(ctx, defaultSecret)
	} else if err != nil {
		log.Error(err, "Failed to get "+rhOperatorPullSecretName)
		return err
	}
	return nil
}

// CreateEmptyConfigMap creates an empty ConfigMap that is used by CNO (Cluster Network Operator) to
// inject custom CA into containers
func CreateEmptyConfigMap(r client.Client, scheme *runtime.Scheme, ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp, log logr.Logger) (ctrl.Result, error) {

	configMap := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: caConfigMapName, Namespace: pulp.Namespace}, configMap)

	expected_cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      caConfigMapName,
			Namespace: pulp.Namespace,
			Labels: map[string]string{
				"config.openshift.io/inject-trusted-cabundle": "true",
			},
		},
		Data: map[string]string{},
	}

	// create the configmap if not found
	if err != nil && k8s_errors.IsNotFound(err) {
		log.V(1).Info("Creating a new empty ConfigMap")
		ctrl.SetControllerReference(pulp, expected_cm, scheme)
		err = r.Create(ctx, expected_cm)
		if err != nil {
			log.Error(err, "Failed to create empty ConfigMap")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get empty ConfigMap")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// mountCASpec adds the trusted-ca bundle into []volume and []volumeMount if pulp.Spec.TrustedCA is true
func mountCASpec(pulp *repomanagerpulpprojectorgv1beta2.Pulp, volumes []corev1.Volume, volumeMounts []corev1.VolumeMount) ([]corev1.Volume, []corev1.VolumeMount) {

	if pulp.Spec.TrustedCa {

		// trustedCAVolume contains the configmap with the custom ca bundle
		trustedCAVolume := corev1.Volume{
			Name: "trusted-ca",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: caConfigMapName,
					},
					Items: []corev1.KeyToPath{
						{Key: "ca-bundle.crt", Path: "tls-ca-bundle.pem"},
					},
				},
			},
		}
		volumes = append(volumes, trustedCAVolume)

		// trustedCAMount defines the mount point of the configmap
		// with the custom ca bundle
		trustedCAMount := corev1.VolumeMount{
			Name:      "trusted-ca",
			MountPath: "/etc/pki/ca-trust/extracted/pem",
			ReadOnly:  true,
		}
		volumeMounts = append(volumeMounts, trustedCAMount)
	}

	return volumes, volumeMounts
}

// GetRouteHost defines route host based on ingress default cluster domain if no .spec.route_host defined
func GetRouteHost(ctx context.Context, client client.Client, pulp *repomanagerpulpprojectorgv1beta2.Pulp) string {
	routeHost := pulp.Spec.RouteHost
	if len(pulp.Spec.RouteHost) == 0 {
		ingress := &configv1.Ingress{}
		client.Get(ctx, types.NamespacedName{Name: "cluster"}, ingress)
		routeHost = pulp.Name + "." + ingress.Spec.Domain
	}
	return routeHost
}

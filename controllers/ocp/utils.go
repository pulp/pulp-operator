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
	"strings"

	"github.com/go-logr/logr"
	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	"github.com/pulp/pulp-operator/controllers"
	"github.com/pulp/pulp-operator/controllers/settings"
	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateRHOperatorPullSecret creates a default secret called redhat-operators-pull-secret
func CreateRHOperatorPullSecret(r client.Client, ctx context.Context, pulp pulpv1.Pulp) error {
	log := logr.Logger{}

	pulpName := pulp.Name
	namespace := pulp.Namespace

	secretName := settings.RedHatOperatorPullSecret(pulpName)
	// Get redhat-operators-pull-secret
	defaultSecret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, defaultSecret)

	// Create the secret in case it is not found
	if err != nil && k8s_errors.IsNotFound(err) {
		defaultSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: namespace,
				Labels:    settings.CommonLabels(pulp),
			},
			StringData: map[string]string{
				"operator": "pulp",
			},
		}
		r.Create(ctx, defaultSecret)
	} else if err != nil {
		log.Error(err, "Failed to get "+secretName)
		return err
	}
	return nil
}

// CreateEmptyConfigMap creates an empty ConfigMap that is used by CNO (Cluster Network Operator) to
// inject custom CA into containers
func CreateEmptyConfigMap(r client.Client, scheme *runtime.Scheme, ctx context.Context, pulp *pulpv1.Pulp, log logr.Logger) (ctrl.Result, error) {

	configMapName := settings.EmptyCAConfigMapName(pulp.Name)
	configMap := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: pulp.Namespace}, configMap)

	labels := settings.CommonLabels(*pulp)
	labels["config.openshift.io/inject-trusted-cabundle"] = "true"
	expected_cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: pulp.Namespace,
			Labels:    labels,
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

// MountCASpec adds the trusted-ca bundle into []volume and []volumeMount if pulp.Spec.TrustedCA is true
// On OpenShift: uses the operator-created ConfigMap with CNO injection
// On vanilla K8s: uses a user-specified ConfigMap (which can be managed manually or by trust-manager)
func MountCASpec(pulp *pulpv1.Pulp, volumes []corev1.Volume, volumeMounts []corev1.VolumeMount) ([]corev1.Volume, []corev1.VolumeMount) {

	if pulp.Spec.TrustedCa {
		var configMapName string
		var configMapKey string

		// Determine ConfigMap name and key based on configuration
		if len(pulp.Spec.TrustedCaConfigMapKey) > 0 {
			// Vanilla K8s mode: parse "configmap-name:key" format
			// If no separator, assume it's just the ConfigMap name and use the first key in the map
			parts := strings.Split(pulp.Spec.TrustedCaConfigMapKey, ":")
			if len(parts) == 2 {
				configMapName = parts[0]
				configMapKey = parts[1]
			} else {
				// Just ConfigMap name provided, use empty key to get first key in map
				configMapName = parts[0]
				configMapKey = ""
			}
		} else {
			// OpenShift mode: use the operator-created ConfigMap with CNO injection
			configMapName = settings.EmptyCAConfigMapName(pulp.Name)
			configMapKey = "ca-bundle.crt"
		}

		// trustedCAVolume contains the configmap with the custom ca bundle
		defaultMode := int32(420)
		configMapVolumeSource := &corev1.ConfigMapVolumeSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: configMapName,
			},
			DefaultMode: &defaultMode,
		}

		// If a specific key is provided, map it to the expected path
		// Otherwise, mount all keys from the ConfigMap (first key will be used)
		if configMapKey != "" {
			configMapVolumeSource.Items = []corev1.KeyToPath{
				{Key: configMapKey, Path: "tls-ca-bundle.pem"},
			}
		}

		trustedCAVolume := corev1.Volume{
			Name: "trusted-ca",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: configMapVolumeSource,
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
func GetRouteHost(pulp *pulpv1.Pulp) string {
	if len(pulp.Spec.RouteHost) == 0 {
		controllers.CustomZapLogger().Warn(`ingress_type defined as "route" but no route_host provided.`)
		controllers.CustomZapLogger().Warn(`Setting "example.com" as the default hostname for routes ...`)
		return "example.com"
	}

	return pulp.Spec.RouteHost
}

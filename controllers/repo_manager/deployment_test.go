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

package repo_manager

import (
	"testing"

	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// TestDefaultsForVanillaDeployment_NoTrustManager verifies no changes when trust-manager is not configured
func TestDefaultsForVanillaDeployment_NoTrustManager(t *testing.T) {
	pulp := &pulpv1.Pulp{
		Spec: pulpv1.PulpSpec{
			TrustedCa:             false,
			TrustedCaConfigMapKey: "",
		},
	}
	pulp.Name = "test-pulp"

	deployment := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{Name: "existing-volume"},
					},
					Containers: []corev1.Container{
						{
							Name: "pulp",
							VolumeMounts: []corev1.VolumeMount{
								{Name: "existing-mount"},
							},
						},
					},
				},
			},
		},
	}

	defaultsForVanillaDeployment(deployment, pulp)

	// Should still have only the original volume and mount
	if len(deployment.Spec.Template.Spec.Volumes) != 1 {
		t.Errorf("Expected 1 volume when trust-manager not configured, got %d", len(deployment.Spec.Template.Spec.Volumes))
	}

	if len(deployment.Spec.Template.Spec.Containers[0].VolumeMounts) != 1 {
		t.Errorf("Expected 1 volumeMount when trust-manager not configured, got %d", len(deployment.Spec.Template.Spec.Containers[0].VolumeMounts))
	}
}

// TestDefaultsForVanillaDeployment_TrustedCaEnabledNoKey verifies error handling when TrustedCa is set but ConfigMapKey is not
func TestDefaultsForVanillaDeployment_TrustedCaEnabledNoKey(t *testing.T) {
	pulp := &pulpv1.Pulp{
		Spec: pulpv1.PulpSpec{
			TrustedCa:             true,
			TrustedCaConfigMapKey: "", // No key - invalid configuration on vanilla K8s
		},
	}
	pulp.Name = "test-pulp"

	deployment := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{Name: "existing-volume"},
					},
					Containers: []corev1.Container{
						{
							Name: "pulp",
							VolumeMounts: []corev1.VolumeMount{
								{Name: "existing-mount"},
							},
						},
					},
				},
			},
		},
	}

	// This should log an error and return early without modifying the deployment
	defaultsForVanillaDeployment(deployment, pulp)

	// Should still have only the original volume and mount (invalid config, no changes made)
	if len(deployment.Spec.Template.Spec.Volumes) != 1 {
		t.Errorf("Expected 1 volume when ConfigMapKey not set (invalid config), got %d", len(deployment.Spec.Template.Spec.Volumes))
	}

	if len(deployment.Spec.Template.Spec.Containers[0].VolumeMounts) != 1 {
		t.Errorf("Expected 1 volumeMount when ConfigMapKey not set (invalid config), got %d", len(deployment.Spec.Template.Spec.Containers[0].VolumeMounts))
	}
}

// TestDefaultsForVanillaDeployment_WithTrustManager verifies CA mounting when trust-manager is configured
func TestDefaultsForVanillaDeployment_WithTrustManager(t *testing.T) {
	pulpName := "test-pulp"
	configMapName := "vault-ca-defaults-bundle"
	configMapKey := "ca.crt"
	separatorFormat := configMapName + ":" + configMapKey

	pulp := &pulpv1.Pulp{
		Spec: pulpv1.PulpSpec{
			TrustedCa:             true,
			TrustedCaConfigMapKey: separatorFormat,
		},
	}
	pulp.Name = pulpName

	deployment := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{Name: "existing-volume"},
					},
					Containers: []corev1.Container{
						{
							Name: "pulp",
							VolumeMounts: []corev1.VolumeMount{
								{Name: "existing-mount"},
							},
						},
					},
				},
			},
		},
	}

	defaultsForVanillaDeployment(deployment, pulp)

	// Should now have 2 volumes (existing + trusted-ca)
	if len(deployment.Spec.Template.Spec.Volumes) != 2 {
		t.Fatalf("Expected 2 volumes with trust-manager configured, got %d", len(deployment.Spec.Template.Spec.Volumes))
	}

	// Should now have 2 mounts (existing + trusted-ca)
	if len(deployment.Spec.Template.Spec.Containers[0].VolumeMounts) != 2 {
		t.Fatalf("Expected 2 volumeMounts with trust-manager configured, got %d", len(deployment.Spec.Template.Spec.Containers[0].VolumeMounts))
	}

	// Verify the CA volume was added correctly
	caVolume := deployment.Spec.Template.Spec.Volumes[1]
	if caVolume.Name != "trusted-ca" {
		t.Errorf("Expected volume name 'trusted-ca', got '%s'", caVolume.Name)
	}

	if caVolume.ConfigMap.Name != configMapName {
		t.Errorf("Expected ConfigMap name '%s', got '%s'", configMapName, caVolume.ConfigMap.Name)
	}

	if len(caVolume.ConfigMap.Items) != 1 {
		t.Fatalf("Expected 1 item in ConfigMap volume, got %d", len(caVolume.ConfigMap.Items))
	}

	if caVolume.ConfigMap.Items[0].Key != configMapKey {
		t.Errorf("Expected ConfigMap key '%s', got '%s'", configMapKey, caVolume.ConfigMap.Items[0].Key)
	}

	// Verify the CA mount was added correctly
	caMount := deployment.Spec.Template.Spec.Containers[0].VolumeMounts[1]
	if caMount.Name != "trusted-ca" {
		t.Errorf("Expected volumeMount name 'trusted-ca', got '%s'", caMount.Name)
	}

	if caMount.MountPath != "/etc/pki/ca-trust/extracted/pem" {
		t.Errorf("Expected mountPath '/etc/pki/ca-trust/extracted/pem', got '%s'", caMount.MountPath)
	}
}

// Note: Full integration tests with Deploy() require additional setup
// (k8s client, schemes, etc.) and are covered by the controller_test.go
// integration test suite. These unit tests focus on the defaultsForVanillaDeployment
// function behavior in isolation.

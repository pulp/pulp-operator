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
	"testing"

	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	"github.com/pulp/pulp-operator/controllers"
	"github.com/pulp/pulp-operator/controllers/settings"
	corev1 "k8s.io/api/core/v1"
)

// TestMountCASpec_Disabled verifies that no volumes are added when TrustedCa is false
func TestMountCASpec_Disabled(t *testing.T) {
	pulp := &pulpv1.Pulp{
		Spec: pulpv1.PulpSpec{
			TrustedCa: false,
		},
	}

	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}

	resultVolumes := controllers.SetCAVolumes(pulp, volumes)
	resultVolumeMounts := controllers.SetCAVolumeMounts(pulp, volumeMounts)

	if len(resultVolumes) != 0 {
		t.Errorf("Expected 0 volumes when TrustedCa is false, got %d", len(resultVolumes))
	}

	if len(resultVolumeMounts) != 0 {
		t.Errorf("Expected 0 volumeMounts when TrustedCa is false, got %d", len(resultVolumeMounts))
	}
}

// TestMountCASpec_OpenShiftMode verifies OpenShift mode (no trust-manager key specified)
func TestMountCASpec_OpenShiftMode(t *testing.T) {
	pulpName := "test-pulp"
	pulp := &pulpv1.Pulp{
		Spec: pulpv1.PulpSpec{
			TrustedCa:             true,
			TrustedCaConfigMapKey: nil, // Empty means OpenShift mode
		},
	}
	pulp.Name = pulpName

	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}

	resultVolumes := controllers.SetCAVolumes(pulp, volumes)
	resultVolumeMounts := controllers.SetCAVolumeMounts(pulp, volumeMounts)

	// Verify we got exactly one volume and one mount
	if len(resultVolumes) != 1 {
		t.Fatalf("Expected 1 volume in OpenShift mode, got %d", len(resultVolumes))
	}

	if len(resultVolumeMounts) != 1 {
		t.Fatalf("Expected 1 volumeMount in OpenShift mode, got %d", len(resultVolumeMounts))
	}

	// Verify volume configuration
	volume := resultVolumes[0]
	if volume.Name != "trusted-ca" {
		t.Errorf("Expected volume name 'trusted-ca', got '%s'", volume.Name)
	}

	expectedConfigMapName := settings.EmptyCAConfigMapName(pulpName)
	if volume.ConfigMap.Name != expectedConfigMapName {
		t.Errorf("Expected ConfigMap name '%s', got '%s'", expectedConfigMapName, volume.ConfigMap.Name)
	}

	if len(volume.ConfigMap.Items) != 1 {
		t.Fatalf("Expected 1 item in ConfigMap volume, got %d", len(volume.ConfigMap.Items))
	}

	if volume.ConfigMap.Items[0].Key != "ca-bundle.crt" {
		t.Errorf("Expected ConfigMap key 'ca-bundle.crt', got '%s'", volume.ConfigMap.Items[0].Key)
	}

	if volume.ConfigMap.Items[0].Path != "tls-ca-bundle.pem" {
		t.Errorf("Expected path 'tls-ca-bundle.pem', got '%s'", volume.ConfigMap.Items[0].Path)
	}

	// Verify volume mount configuration
	volumeMount := resultVolumeMounts[0]
	if volumeMount.Name != "trusted-ca" {
		t.Errorf("Expected volumeMount name 'trusted-ca', got '%s'", volumeMount.Name)
	}

	if volumeMount.MountPath != "/etc/pki/ca-trust/extracted/pem" {
		t.Errorf("Expected mountPath '/etc/pki/ca-trust/extracted/pem', got '%s'", volumeMount.MountPath)
	}

	if !volumeMount.ReadOnly {
		t.Error("Expected volumeMount to be ReadOnly")
	}
}

// TestMountCASpec_TrustManagerMode verifies trust-manager mode with just ConfigMap name (no key specified)
func TestMountCASpec_TrustManagerMode(t *testing.T) {
	pulpName := "test-pulp"
	configMapName := "my-ca-bundle"

	pulp := &pulpv1.Pulp{
		Spec: pulpv1.PulpSpec{
			TrustedCa:             true,
			TrustedCaConfigMapKey: &configMapName, // Just ConfigMap name, no ":"
		},
	}
	pulp.Name = pulpName

	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}

	resultVolumes := controllers.SetCAVolumes(pulp, volumes)
	resultVolumeMounts := controllers.SetCAVolumeMounts(pulp, volumeMounts)

	// Verify we got exactly one volume and one mount
	if len(resultVolumes) != 1 {
		t.Fatalf("Expected 1 volume in trust-manager mode, got %d", len(resultVolumes))
	}

	if len(resultVolumeMounts) != 1 {
		t.Fatalf("Expected 1 volumeMount in trust-manager mode, got %d", len(resultVolumeMounts))
	}

	// Verify volume configuration
	volume := resultVolumes[0]
	if volume.Name != "trusted-ca" {
		t.Errorf("Expected volume name 'trusted-ca', got '%s'", volume.Name)
	}

	if volume.ConfigMap.Name != configMapName {
		t.Errorf("Expected ConfigMap name '%s', got '%s'", configMapName, volume.ConfigMap.Name)
	}

	// When no key is specified, Items should be empty (mounts all keys)
	if len(volume.ConfigMap.Items) != 0 {
		t.Errorf("Expected 0 items in ConfigMap volume (mount all keys), got %d", len(volume.ConfigMap.Items))
	}

	// Verify volume mount configuration
	volumeMount := resultVolumeMounts[0]
	if volumeMount.Name != "trusted-ca" {
		t.Errorf("Expected volumeMount name 'trusted-ca', got '%s'", volumeMount.Name)
	}

	if volumeMount.MountPath != "/etc/pki/ca-trust/extracted/pem" {
		t.Errorf("Expected mountPath '/etc/pki/ca-trust/extracted/pem', got '%s'", volumeMount.MountPath)
	}

	if !volumeMount.ReadOnly {
		t.Error("Expected volumeMount to be ReadOnly")
	}
}

// TestMountCASpec_CustomKey verifies trust-manager mode with "configmap:key" format
func TestMountCASpec_CustomKey(t *testing.T) {
	pulpName := "test-pulp"
	configMapName := "my-custom-bundle"
	customKey := "custom-ca-bundle.pem"
	separatorFormat := configMapName + ":" + customKey

	pulp := &pulpv1.Pulp{
		Spec: pulpv1.PulpSpec{
			TrustedCa:             true,
			TrustedCaConfigMapKey: &separatorFormat,
		},
	}
	pulp.Name = pulpName

	volumes := []corev1.Volume{}
	resultVolumes := controllers.SetCAVolumes(pulp, volumes)

	if len(resultVolumes) != 1 {
		t.Fatalf("Expected 1 volume, got %d", len(resultVolumes))
	}

	volume := resultVolumes[0]
	if volume.ConfigMap.Name != configMapName {
		t.Errorf("Expected ConfigMap name '%s', got '%s'", configMapName, volume.ConfigMap.Name)
	}

	if len(volume.ConfigMap.Items) != 1 {
		t.Fatalf("Expected 1 item in ConfigMap volume, got %d", len(volume.ConfigMap.Items))
	}

	if volume.ConfigMap.Items[0].Key != customKey {
		t.Errorf("Expected ConfigMap key '%s', got '%s'", customKey, volume.ConfigMap.Items[0].Key)
	}
}

// TestMountCASpec_PreservesExistingVolumes verifies that existing volumes are preserved
func TestMountCASpec_PreservesExistingVolumes(t *testing.T) {
	configMapName := "my-bund:ca-bundle.crt"
	pulp := &pulpv1.Pulp{
		Spec: pulpv1.PulpSpec{
			TrustedCa:             true,
			TrustedCaConfigMapKey: &configMapName,
		},
	}
	pulp.Name = "test-pulp"

	// Start with existing volumes and mounts
	existingVolumes := []corev1.Volume{
		{Name: "existing-volume-1"},
		{Name: "existing-volume-2"},
	}
	existingVolumeMounts := []corev1.VolumeMount{
		{Name: "existing-mount-1"},
		{Name: "existing-mount-2"},
	}

	resultVolumes := controllers.SetCAVolumes(pulp, existingVolumes)
	resultVolumeMounts := controllers.SetCAVolumeMounts(pulp, existingVolumeMounts)

	// Should have 3 volumes (2 existing + 1 new)
	if len(resultVolumes) != 3 {
		t.Errorf("Expected 3 volumes (2 existing + 1 new), got %d", len(resultVolumes))
	}

	// Should have 3 mounts (2 existing + 1 new)
	if len(resultVolumeMounts) != 3 {
		t.Errorf("Expected 3 volumeMounts (2 existing + 1 new), got %d", len(resultVolumeMounts))
	}

	// Verify existing volumes are preserved
	if resultVolumes[0].Name != "existing-volume-1" {
		t.Error("First existing volume was not preserved")
	}
	if resultVolumes[1].Name != "existing-volume-2" {
		t.Error("Second existing volume was not preserved")
	}
	if resultVolumes[2].Name != "trusted-ca" {
		t.Error("New trusted-ca volume not added correctly")
	}

	// Verify existing mounts are preserved
	if resultVolumeMounts[0].Name != "existing-mount-1" {
		t.Error("First existing mount was not preserved")
	}
	if resultVolumeMounts[1].Name != "existing-mount-2" {
		t.Error("Second existing mount was not preserved")
	}
	if resultVolumeMounts[2].Name != "trusted-ca" {
		t.Error("New trusted-ca mount not added correctly")
	}
}

// TestMountCASpec_SeparatorFormat verifies the "configmap-name:key" separator format
func TestMountCASpec_SeparatorFormat(t *testing.T) {
	pulpName := "test-pulp"
	configMapName := "vault-ca-defaults-bundle"
	configMapKey := "ca.crt"
	separatorFormat := configMapName + ":" + configMapKey

	pulp := &pulpv1.Pulp{
		Spec: pulpv1.PulpSpec{
			TrustedCa:             true,
			TrustedCaConfigMapKey: &separatorFormat,
		},
	}
	pulp.Name = pulpName

	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}

	resultVolumes := controllers.SetCAVolumes(pulp, volumes)
	resultVolumeMounts := controllers.SetCAVolumeMounts(pulp, volumeMounts)

	// Verify we got exactly one volume and one mount
	if len(resultVolumes) != 1 {
		t.Fatalf("Expected 1 volume with separator format, got %d", len(resultVolumes))
	}

	if len(resultVolumeMounts) != 1 {
		t.Fatalf("Expected 1 volumeMount with separator format, got %d", len(resultVolumeMounts))
	}

	// Verify volume configuration - should use the parsed ConfigMap name
	volume := resultVolumes[0]
	if volume.Name != "trusted-ca" {
		t.Errorf("Expected volume name 'trusted-ca', got '%s'", volume.Name)
	}

	if volume.ConfigMap.Name != configMapName {
		t.Errorf("Expected ConfigMap name '%s', got '%s'", configMapName, volume.ConfigMap.Name)
	}

	if len(volume.ConfigMap.Items) != 1 {
		t.Fatalf("Expected 1 item in ConfigMap volume, got %d", len(volume.ConfigMap.Items))
	}

	if volume.ConfigMap.Items[0].Key != configMapKey {
		t.Errorf("Expected ConfigMap key '%s', got '%s'", configMapKey, volume.ConfigMap.Items[0].Key)
	}

	if volume.ConfigMap.Items[0].Path != "tls-ca-bundle.pem" {
		t.Errorf("Expected path 'tls-ca-bundle.pem', got '%s'", volume.ConfigMap.Items[0].Path)
	}

	// Verify volume mount configuration
	volumeMount := resultVolumeMounts[0]
	if volumeMount.Name != "trusted-ca" {
		t.Errorf("Expected volumeMount name 'trusted-ca', got '%s'", volumeMount.Name)
	}

	if volumeMount.MountPath != "/etc/pki/ca-trust/extracted/pem" {
		t.Errorf("Expected mountPath '/etc/pki/ca-trust/extracted/pem', got '%s'", volumeMount.MountPath)
	}

	if !volumeMount.ReadOnly {
		t.Error("Expected volumeMount to be ReadOnly")
	}
}

// TestMountCASpec_MountPathConsistency verifies mount path is consistent across modes
func TestMountCASpec_MountPathConsistency(t *testing.T) {
	pulpName := "test-pulp"
	expectedMountPath := "/etc/pki/ca-trust/extracted/pem"

	testCases := []struct {
		name                  string
		trustedCa             bool
		trustedCaConfigMapKey string
		description           string
	}{
		{
			name:                  "OpenShift mode",
			trustedCa:             true,
			trustedCaConfigMapKey: "",
			description:           "OpenShift CNO injection mode",
		},
		{
			name:                  "trust-manager mode - ConfigMap name only",
			trustedCa:             true,
			trustedCaConfigMapKey: "my-ca-bundle",
			description:           "trust-manager mode with ConfigMap name only",
		},
		{
			name:                  "trust-manager mode - with separator",
			trustedCa:             true,
			trustedCaConfigMapKey: "my-ca-bundle:ca-bundle.crt",
			description:           "trust-manager mode with configmap:key format",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pulp := &pulpv1.Pulp{
				Spec: pulpv1.PulpSpec{
					TrustedCa:             tc.trustedCa,
					TrustedCaConfigMapKey: &tc.trustedCaConfigMapKey,
				},
			}
			pulp.Name = pulpName

			resultVolumeMounts := controllers.SetCAVolumeMounts(pulp, []corev1.VolumeMount{})

			if len(resultVolumeMounts) != 1 {
				t.Fatalf("Expected 1 volumeMount, got %d", len(resultVolumeMounts))
			}

			if resultVolumeMounts[0].MountPath != expectedMountPath {
				t.Errorf("%s: Expected mountPath '%s', got '%s'",
					tc.description, expectedMountPath, resultVolumeMounts[0].MountPath)
			}
		})
	}
}

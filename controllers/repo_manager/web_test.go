/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package repo_manager

import (
	"context"
	"strings"
	"testing"

	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	"github.com/pulp/pulp-operator/controllers"
	"github.com/pulp/pulp-operator/controllers/settings"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// TestPulpWebContentPathPrefixPropagation verifies that overriding
// CONTENT_PATH_PREFIX via a custom_pulp_settings ConfigMap:
//  1. Updates the rendered nginx.conf in the pulp-web ConfigMap.
//  2. Shifts the pulp-web pod-template hash annotation so the Deployment
//     gets rolled (necessary because nginx.conf is mounted via subPath and
//     kubelet does not auto-update those mounts).
func TestPulpWebContentPathPrefixPropagation(t *testing.T) {
	const (
		pulpName       = "test-pulp"
		pulpNamespace  = "default"
		customCMName   = "pulp-custom-settings"
		defaultPath    = "/pulp/content/"
		overriddenPath = "/pulp/repos/"
		annotationKey  = "repo-manager.pulpproject.org/web-config-hash"
	)

	scheme := runtime.NewScheme()
	if err := pulpv1.AddToScheme(scheme); err != nil {
		t.Fatalf("add pulp scheme: %v", err)
	}
	if err := appsv1.AddToScheme(scheme); err != nil {
		t.Fatalf("add apps scheme: %v", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add core scheme: %v", err)
	}

	pulp := &pulpv1.Pulp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pulpName,
			Namespace: pulpNamespace,
		},
		Spec: pulpv1.PulpSpec{
			CustomPulpSettings: customCMName,
			Web:                pulpv1.Web{Replicas: 1},
		},
	}

	render := func(t *testing.T, contentPathPrefix string) (string, string) {
		t.Helper()

		customCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      customCMName,
				Namespace: pulpNamespace,
			},
			Data: map[string]string{
				// Quoted so the operator writes a valid Python string literal
				// into settings.py — same shape a user would use.
				"content_path_prefix": `"` + contentPathPrefix + `"`,
			},
		}

		client := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(pulp.DeepCopy(), customCM).
			Build()

		r := &RepoManagerReconciler{
			Client:    client,
			Scheme:    scheme,
			RawLogger: zap.New(zap.UseDevMode(true)),
		}
		funcResources := controllers.FunctionResources{
			Context: context.Background(),
			Client:  client,
			Pulp:    pulp,
			Scheme:  scheme,
			Logger:  zap.New(zap.UseDevMode(true)),
		}

		cm := r.pulpWebConfigMap(context.Background(), pulp)
		if cm.Name != settings.PulpWebConfigMapName(pulpName) {
			t.Fatalf("unexpected ConfigMap name: %q", cm.Name)
		}
		nginxConf, ok := cm.Data["nginx.conf"]
		if !ok {
			t.Fatalf("nginx.conf key missing from rendered ConfigMap")
		}

		dep := r.deploymentForPulpWeb(pulp, funcResources, cm)
		hash := dep.Spec.Template.Annotations[annotationKey]
		if hash == "" {
			t.Fatalf("pulp-web pod template missing %q annotation", annotationKey)
		}

		return nginxConf, hash
	}

	defaultConf, defaultHash := render(t, defaultPath)
	overriddenConf, overriddenHash := render(t, overriddenPath)

	// 1. nginx.conf reflects the override.
	if !strings.Contains(defaultConf, "location "+defaultPath+" {") {
		t.Errorf("default nginx.conf is missing %q location block:\n%s", defaultPath, defaultConf)
	}
	if !strings.Contains(overriddenConf, "location "+overriddenPath+" {") {
		t.Errorf("overridden nginx.conf is missing %q location block:\n%s", overriddenPath, overriddenConf)
	}
	if strings.Contains(overriddenConf, "location "+defaultPath+" {") {
		t.Errorf("overridden nginx.conf still contains default %q location block:\n%s", defaultPath, overriddenConf)
	}

	// 2. The pod-template hash annotation changed, so the Deployment Spec
	// hash changes and CheckDeploymentSpec will roll the pulp-web pods.
	if defaultHash == overriddenHash {
		t.Errorf("pod-template %q annotation did not change between configs (%q); "+
			"pulp-web pods would not roll when CONTENT_PATH_PREFIX is updated",
			annotationKey, defaultHash)
	}
}

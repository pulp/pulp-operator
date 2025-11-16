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

package controllers

import (
	"testing"

	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	"github.com/pulp/pulp-operator/controllers/settings"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestCheckDeploymentSpecWithHPA verifies that when HPA is enabled,
// the replicas field is excluded from drift detection
func TestCheckDeploymentSpecWithHPA(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = pulpv1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	tests := []struct {
		name               string
		hpaEnabled         bool
		expectedReplicas   *int32
		currentReplicas    *int32
		shouldDetectChange bool
		description        string
	}{
		{
			name:               "HPA enabled - different replicas should NOT trigger reconciliation",
			hpaEnabled:         true,
			expectedReplicas:   nil,
			currentReplicas:    int32Ptr(3),
			shouldDetectChange: false,
			description:        "When HPA is enabled, operator should ignore replica count differences",
		},
		{
			name:               "HPA disabled - different replicas SHOULD trigger reconciliation",
			hpaEnabled:         false,
			expectedReplicas:   int32Ptr(2),
			currentReplicas:    int32Ptr(3),
			shouldDetectChange: true,
			description:        "When HPA is disabled, operator should detect replica count differences",
		},
		{
			name:               "HPA enabled - same replicas should NOT trigger reconciliation",
			hpaEnabled:         true,
			expectedReplicas:   nil,
			currentReplicas:    nil,
			shouldDetectChange: false,
			description:        "When HPA is enabled and replicas are both nil, no change detected",
		},
		{
			name:               "HPA disabled - same replicas should NOT trigger reconciliation",
			hpaEnabled:         false,
			expectedReplicas:   int32Ptr(2),
			currentReplicas:    int32Ptr(2),
			shouldDetectChange: false,
			description:        "When HPA is disabled and replicas match, no change detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create Pulp CR with HPA configuration
			pulp := &pulpv1.Pulp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pulp",
					Namespace: "test-namespace",
				},
				Spec: pulpv1.PulpSpec{
					Api: pulpv1.Api{
						Replicas: 2,
					},
				},
			}

			if tt.hpaEnabled {
				pulp.Spec.Api.HPA = &pulpv1.HPA{
					Enabled:     true,
					MaxReplicas: 10,
				}
			}

			client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pulp).Build()

			// Create expected deployment
			expectedDep := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      settings.API.DeploymentName(pulp.Name),
					Namespace: pulp.Namespace,
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: tt.expectedReplicas,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "pulp-api"},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": "pulp-api"},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "api",
									Image: "test:latest",
								},
							},
						},
					},
				},
			}

			// Create current deployment (simulating HPA has changed replicas)
			currentDep := expectedDep.DeepCopy()
			currentDep.Spec.Replicas = tt.currentReplicas

			// Calculate and set the hash label on current deployment
			// This simulates what happens in production where the deployment has the hash label
			expectedHash := CalculateHash(expectedDep.Spec)
			SetHashLabel(expectedHash, currentDep)

			// Create FunctionResources
			funcResources := FunctionResources{
				Client: client,
				Pulp:   pulp,
				Scheme: scheme,
			}

			// Test the CheckDeploymentSpec function
			changed := CheckDeploymentSpec(*expectedDep, *currentDep, funcResources)

			if changed != tt.shouldDetectChange {
				t.Errorf("%s: expected change detection = %v, got %v. %s",
					tt.name, tt.shouldDetectChange, changed, tt.description)
			}
		})
	}
}

// TestIsHPAManagedDeployment verifies the HPA detection logic
func TestIsHPAManagedDeployment(t *testing.T) {
	tests := []struct {
		name           string
		deploymentName string
		hpaConfig      *pulpv1.HPA
		component      settings.PulpcoreType
		expected       bool
	}{
		{
			name:      "API deployment with HPA enabled",
			component: settings.API,
			hpaConfig: &pulpv1.HPA{Enabled: true, MaxReplicas: 10},
			expected:  true,
		},
		{
			name:      "API deployment with HPA disabled",
			component: settings.API,
			hpaConfig: &pulpv1.HPA{Enabled: false, MaxReplicas: 10},
			expected:  false,
		},
		{
			name:      "API deployment without HPA config",
			component: settings.API,
			hpaConfig: nil,
			expected:  false,
		},
		{
			name:      "Content deployment with HPA enabled",
			component: settings.CONTENT,
			hpaConfig: &pulpv1.HPA{Enabled: true, MaxReplicas: 10},
			expected:  true,
		},
		{
			name:      "Worker deployment with HPA enabled",
			component: settings.WORKER,
			hpaConfig: &pulpv1.HPA{Enabled: true, MaxReplicas: 10},
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pulp := &pulpv1.Pulp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pulp",
					Namespace: "test-namespace",
				},
				Spec: pulpv1.PulpSpec{},
			}

			// Set HPA config for the specific component
			switch tt.component {
			case settings.API:
				pulp.Spec.Api.HPA = tt.hpaConfig
			case settings.CONTENT:
				pulp.Spec.Content.HPA = tt.hpaConfig
			case settings.WORKER:
				pulp.Spec.Worker.HPA = tt.hpaConfig
			}

			deploymentName := tt.component.DeploymentName(pulp.Name)
			result := isHPAManagedDeployment(deploymentName, pulp)

			if result != tt.expected {
				t.Errorf("%s: expected %v, got %v", tt.name, tt.expected, result)
			}
		})
	}
}

func int32Ptr(i int32) *int32 {
	return &i
}

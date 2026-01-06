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
	"context"
	"testing"

	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	"github.com/pulp/pulp-operator/controllers/settings"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// TestDeprecatedServiceAccountField verifies that the deprecated serviceAccount field
// is properly stripped during hash calculation to prevent false positives
func TestDeprecatedServiceAccountField(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = pulpv1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	pulp := &pulpv1.Pulp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pulp",
			Namespace: "test-namespace",
		},
		Spec: pulpv1.PulpSpec{
			Api: pulpv1.Api{
				Replicas: 1,
			},
		},
	}

	serviceAccountName := settings.PulpServiceAccount(pulp.Name)

	tests := []struct {
		name                            string
		currentHasDeprecatedServiceAcct bool
		shouldDetectChange              bool
		description                     string
	}{
		{
			name:                            "Kubernetes populates DeprecatedServiceAccount - should NOT detect change",
			currentHasDeprecatedServiceAcct: true,
			shouldDetectChange:              false,
			description:                     "When Kubernetes populates DeprecatedServiceAccount, it's stripped during comparison",
		},
		{
			name:                            "Current without DeprecatedServiceAccount - should NOT detect change",
			currentHasDeprecatedServiceAcct: false,
			shouldDetectChange:              false,
			description:                     "When field is not set, no change detected since we strip it anyway",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pulp).Build()

			// Create expected deployment (operator-created, never has DeprecatedServiceAccount)
			expectedDep := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      settings.API.DeploymentName(pulp.Name),
					Namespace: pulp.Namespace,
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(1),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "pulp-api"},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": "pulp-api"},
						},
						Spec: corev1.PodSpec{
							ServiceAccountName: serviceAccountName,
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

			// Create current deployment - this simulates what Kubernetes API server returns
			currentDep := expectedDep.DeepCopy()
			if tt.currentHasDeprecatedServiceAcct {
				// Kubernetes automatically populates this field from ServiceAccountName
				currentDep.Spec.Template.Spec.DeprecatedServiceAccount = serviceAccountName
			}

			// Create FunctionResources with a logger
			funcResources := FunctionResources{
				Context: context.Background(),
				Client:  client,
				Pulp:    pulp,
				Scheme:  scheme,
				Logger:  zap.New(zap.UseDevMode(true)),
			}

			// Calculate hash with the field stripped and set as label
			// This simulates what AddHashLabel() does
			currentCopy := currentDep.DeepCopy()
			currentCopy.Spec.Template.Spec.DeprecatedServiceAccount = ""
			currentHash := CalculateHash(currentCopy.Spec)
			SetHashLabel(currentHash, currentDep)

			// Test the CheckDeploymentSpec function
			changed := CheckDeploymentSpec(*expectedDep, *currentDep, funcResources)

			if changed != tt.shouldDetectChange {
				t.Errorf("%s: expected change detection = %v, got %v. %s",
					tt.name, tt.shouldDetectChange, changed, tt.description)
			}
		})
	}
}

// TestHashLabelUpdateDuringReconciliation verifies that the hash label is properly
// updated when a deployment is updated, preventing reconciliation loops
func TestHashLabelUpdateDuringReconciliation(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = pulpv1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	pulp := &pulpv1.Pulp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pulp",
			Namespace: "test-namespace",
		},
		Spec: pulpv1.PulpSpec{
			Api: pulpv1.Api{
				Replicas: 1,
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pulp).Build()
	serviceAccountName := settings.PulpServiceAccount(pulp.Name)

	// Create a deployment with an OLD configuration
	oldDep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      settings.API.DeploymentName(pulp.Name),
			Namespace: pulp.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "pulp-api"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "pulp-api"},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: serviceAccountName,
					Containers: []corev1.Container{
						{
							Name:  "api",
							Image: "test:old",
						},
					},
				},
			},
		},
	}

	// Calculate and set hash for OLD deployment (with DeprecatedServiceAccount stripped)
	oldCopy := oldDep.DeepCopy()
	oldCopy.Spec.Template.Spec.DeprecatedServiceAccount = ""
	oldHash := CalculateHash(oldCopy.Spec)
	SetHashLabel(oldHash, oldDep)

	// Create a NEW deployment with updated configuration
	newDep := oldDep.DeepCopy()
	newDep.Spec.Template.Spec.Containers[0].Image = "test:new"

	// This simulates the bug: AddHashLabel was called when the deployment was first created,
	// but the hash label still reflects the OLD spec
	funcResources := FunctionResources{
		Context: context.Background(),
		Client:  client,
		Pulp:    pulp,
		Scheme:  scheme,
		Logger:  zap.New(zap.UseDevMode(true)),
	}

	// Before the fix, this would have the OLD hash
	oldHashFromLabel := GetCurrentHash(newDep)

	// Simulate what happens in updateObject when a change is detected
	// This is the fix: update the hash label before sending to Kubernetes
	AddHashLabel(funcResources, newDep)
	newHashFromLabel := GetCurrentHash(newDep)

	// Calculate what the NEW hash should be (with DeprecatedServiceAccount stripped)
	newCopy := newDep.DeepCopy()
	newCopy.Spec.Template.Spec.DeprecatedServiceAccount = ""
	expectedNewHash := CalculateHash(newCopy.Spec)

	// Verify the hash was updated
	if oldHashFromLabel == newHashFromLabel {
		t.Errorf("Hash label was not updated after AddHashLabel call")
		t.Logf("Old hash: %s, New hash: %s", oldHashFromLabel, newHashFromLabel)
	}

	if newHashFromLabel != expectedNewHash {
		t.Errorf("New hash label doesn't match expected hash")
		t.Logf("Expected: %s, Got: %s", expectedNewHash, newHashFromLabel)
	}

	// Verify that subsequent reconciliation would NOT detect a change
	// (because the hash label now matches the spec)
	currentDep := newDep.DeepCopy()
	changed := CheckDeploymentSpec(*newDep, *currentDep, funcResources)

	if changed {
		t.Errorf("CheckDeploymentSpec detected a change after hash label update, should be stable")
		t.Logf("Hash from label: %s", GetCurrentHash(currentDep))
		currentCopy := currentDep.DeepCopy()
		currentCopy.Spec.Template.Spec.DeprecatedServiceAccount = ""
		t.Logf("Hash from spec: %s", CalculateHash(currentCopy.Spec))
	}
}

// TestReconciliationLoopPrevention verifies that the operator doesn't enter a
// reconciliation loop when deployments are stable
func TestReconciliationLoopPrevention(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = pulpv1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	pulp := &pulpv1.Pulp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pulp",
			Namespace: "test-namespace",
		},
		Spec: pulpv1.PulpSpec{
			Api: pulpv1.Api{
				Replicas: 1,
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pulp).Build()
	serviceAccountName := settings.PulpServiceAccount(pulp.Name)

	// Create a properly configured deployment with DeprecatedServiceAccount
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      settings.API.DeploymentName(pulp.Name),
			Namespace: pulp.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "pulp-api"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "pulp-api"},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:       serviceAccountName,
					DeprecatedServiceAccount: serviceAccountName,
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

	funcResources := FunctionResources{
		Context: context.Background(),
		Client:  client,
		Pulp:    pulp,
		Scheme:  scheme,
		Logger:  zap.New(zap.UseDevMode(true)),
	}

	// Set the hash label
	AddHashLabel(funcResources, dep)

	// Simulate multiple reconciliation loops
	for i := 0; i < 5; i++ {
		currentDep := dep.DeepCopy()
		expectedDep := dep.DeepCopy()

		// Check if deployment has changed
		changed := CheckDeploymentSpec(*expectedDep, *currentDep, funcResources)

		if changed {
			t.Errorf("Reconciliation loop %d detected a change when deployment is stable", i+1)
			t.Logf("Expected hash: %s", HashFromMutated(expectedDep, funcResources))
			t.Logf("Current hash: %s", CalculateHash(currentDep.Spec))
			t.Logf("Hash from label: %s", GetCurrentHash(currentDep))
			break
		}
	}
}

// TestKubernetesAPIMutationHandling verifies that the operator correctly handles
// Kubernetes API server mutations through dry-run
func TestKubernetesAPIMutationHandling(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = pulpv1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	pulp := &pulpv1.Pulp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pulp",
			Namespace: "test-namespace",
		},
		Spec: pulpv1.PulpSpec{
			Api: pulpv1.Api{
				Replicas: 1,
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pulp).Build()

	// Create a deployment WITHOUT Kubernetes defaults
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      settings.API.DeploymentName(pulp.Name),
			Namespace: pulp.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "pulp-api"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "pulp-api"},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:       settings.PulpServiceAccount(pulp.Name),
					DeprecatedServiceAccount: settings.PulpServiceAccount(pulp.Name),
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

	funcResources := FunctionResources{
		Context: context.Background(),
		Client:  client,
		Pulp:    pulp,
		Scheme:  scheme,
		Logger:  zap.New(zap.UseDevMode(true)),
	}

	// Calculate hash before dry-run
	hashBefore := CalculateHash(dep.Spec)

	// HashFromMutated performs a dry-run which simulates what the Kubernetes API would do
	hashAfterMutation := HashFromMutated(dep, funcResources)

	// The hash might be different after mutation due to Kubernetes API defaults
	// But our fix ensures that we calculate the hash AFTER mutation, so it matches
	// what will be in the cluster

	// Verify that when we set this hash on the deployment label,
	// subsequent checks won't detect spurious changes
	SetHashLabel(hashAfterMutation, dep)

	currentDep := dep.DeepCopy()
	changed := CheckDeploymentSpec(*dep, *currentDep, funcResources)

	if changed {
		t.Errorf("Detected spurious change after handling API mutation")
		t.Logf("Hash before mutation: %s", hashBefore)
		t.Logf("Hash after mutation: %s", hashAfterMutation)
		t.Logf("Current hash: %s", CalculateHash(currentDep.Spec))
	}
}

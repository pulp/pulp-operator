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
	"context"
	"testing"

	"github.com/go-logr/logr"
	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	"github.com/pulp/pulp-operator/controllers/settings"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestHpaControllerWebGating verifies which IngressType / IsNginxIngress
// combinations cause hpaController to create an HPA for the pulp-web
// deployment. Regression coverage for the bug where the gate skipped the
// web HPA whenever ingress_type=ingress, even though pulp-web is still
// deployed in that mode unless the ingress controller is nginx-supported.
func TestHpaControllerWebGating(t *testing.T) {
	tests := []struct {
		name           string
		ingressType    string
		isNginxIngress bool
		expectWebHPA   bool
	}{
		{
			name:           "ingress_type=ingress and non-nginx controller deploys pulp-web, so web HPA must be created",
			ingressType:    "ingress",
			isNginxIngress: false,
			expectWebHPA:   true,
		},
		{
			name:           "ingress_type=ingress with nginx controller does NOT deploy pulp-web, so no web HPA",
			ingressType:    "ingress",
			isNginxIngress: true,
			expectWebHPA:   false,
		},
		{
			name:           "ingress_type=route does NOT deploy pulp-web, so no web HPA",
			ingressType:    "route",
			isNginxIngress: false,
			expectWebHPA:   false,
		},
		{
			name:           "ingress_type=nodeport deploys pulp-web, so web HPA must be created",
			ingressType:    "nodeport",
			isNginxIngress: false,
			expectWebHPA:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pulp := newPulpWithHPAEnabled(tt.ingressType, tt.isNginxIngress)

			scheme := runtime.NewScheme()
			if err := pulpv1.AddToScheme(scheme); err != nil {
				t.Fatalf("failed to register pulpv1 scheme: %v", err)
			}
			if err := autoscalingv2.AddToScheme(scheme); err != nil {
				t.Fatalf("failed to register autoscalingv2 scheme: %v", err)
			}

			cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pulp).Build()
			r := &RepoManagerReconciler{Client: cli, Scheme: scheme}

			if _, err := r.hpaController(context.Background(), pulp, logr.Discard()); err != nil {
				t.Fatalf("hpaController returned error: %v", err)
			}

			// API / Content / Worker HPAs are independent of ingress_type
			// and must always be created when enabled. Asserting them here
			// guards against regressions in the non-web branch.
			for _, c := range []settings.PulpcoreType{settings.API, settings.CONTENT, settings.WORKER} {
				if !hpaExists(t, cli, c.DeploymentName(pulp.Name), pulp.Namespace) {
					t.Errorf("expected HPA for %s to be created, but it was not", c)
				}
			}

			webExists := hpaExists(t, cli, settings.WEB.DeploymentName(pulp.Name), pulp.Namespace)
			if webExists != tt.expectWebHPA {
				t.Errorf("web HPA presence: got=%v want=%v", webExists, tt.expectWebHPA)
			}
		})
	}
}

func newPulpWithHPAEnabled(ingressType string, isNginxIngress bool) *pulpv1.Pulp {
	hpa := func() *pulpv1.HPA { return &pulpv1.HPA{Enabled: true, MaxReplicas: 5} }
	return &pulpv1.Pulp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pulp",
			Namespace: "test-namespace",
		},
		Spec: pulpv1.PulpSpec{
			IngressType:    ingressType,
			IsNginxIngress: isNginxIngress,
			Api:            pulpv1.Api{HPA: hpa()},
			Content:        pulpv1.Content{HPA: hpa()},
			Worker:         pulpv1.Worker{HPA: hpa()},
			Web:            pulpv1.Web{HPA: hpa()},
		},
	}
}

func hpaExists(t *testing.T, cli client.Client, name, namespace string) bool {
	t.Helper()
	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	err := cli.Get(context.Background(), types.NamespacedName{Name: name, Namespace: namespace}, hpa)
	if err == nil {
		return true
	}
	if errors.IsNotFound(err) {
		return false
	}
	t.Fatalf("unexpected error fetching HPA %s/%s: %v", namespace, name, err)
	return false
}

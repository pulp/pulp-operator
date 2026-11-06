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

// deploySecurityContextComponent runs the real Deployment generator of a pulpcore component
func deploySecurityContextComponent(t *testing.T, pulp *pulpv1.Pulp, pulpcoreType settings.PulpcoreType) *appsv1.Deployment {
	t.Helper()
	scheme := runtime.NewScheme()
	_ = pulpv1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	resources := FunctionResources{
		Context: context.TODO(),
		Client:  fake.NewClientBuilder().WithScheme(scheme).WithObjects(pulp).Build(),
		Pulp:    pulp,
		Scheme:  scheme,
		Logger:  zap.New(),
	}

	switch pulpcoreType {
	case settings.API:
		return DeploymentAPICommon{}.Deploy(resources).(*appsv1.Deployment)
	case settings.CONTENT:
		return DeploymentContentCommon{}.Deploy(resources).(*appsv1.Deployment)
	default:
		return DeploymentWorkerCommon{}.Deploy(resources).(*appsv1.Deployment)
	}
}

// TestPodSecurityContextDefault verifies that with no override set, every
// component keeps the historical hardcoded runAsUser/fsGroup 700.
func TestPodSecurityContextDefault(t *testing.T) {
	pulp := &pulpv1.Pulp{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pulp", Namespace: "test-namespace"},
		Spec: pulpv1.PulpSpec{
			FileStorageClass: "standard",
			Api:              pulpv1.Api{Replicas: 1},
			Content:          pulpv1.Content{Replicas: 1},
			Worker:           pulpv1.Worker{Replicas: 1},
		},
	}

	for _, pulpcoreType := range []settings.PulpcoreType{settings.API, settings.CONTENT, settings.WORKER} {
		t.Run(string(pulpcoreType), func(t *testing.T) {
			dep := deploySecurityContextComponent(t, pulp, pulpcoreType)
			sc := dep.Spec.Template.Spec.SecurityContext
			if sc == nil || sc.RunAsUser == nil || *sc.RunAsUser != 700 {
				t.Errorf("RunAsUser = %v, want 700", sc)
			}
			if sc == nil || sc.FSGroup == nil || *sc.FSGroup != 700 {
				t.Errorf("FSGroup = %v, want 700", sc)
			}
		})
	}
}

// TestPodSecurityContextOverride verifies that setting pod_security_context per
// component overrides the default and does not leak to other components.
func TestPodSecurityContextOverride(t *testing.T) {
	apiUser := int64(101)
	apiGroup := int64(101)
	workerGroup := int64(2000)
	apiOverride := &corev1.PodSecurityContext{RunAsUser: &apiUser, FSGroup: &apiGroup}
	workerOverride := &corev1.PodSecurityContext{FSGroup: &workerGroup}

	pulp := &pulpv1.Pulp{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pulp", Namespace: "test-namespace"},
		Spec: pulpv1.PulpSpec{
			FileStorageClass: "standard",
			Api:              pulpv1.Api{Replicas: 1, PodSecurityContext: apiOverride},
			Content:          pulpv1.Content{Replicas: 1}, // no override: must keep default
			Worker:           pulpv1.Worker{Replicas: 1, PodSecurityContext: workerOverride},
		},
	}

	apiDep := deploySecurityContextComponent(t, pulp, settings.API)
	apiSC := apiDep.Spec.Template.Spec.SecurityContext
	if apiSC == nil || apiSC.RunAsUser == nil || *apiSC.RunAsUser != 101 {
		t.Errorf("api RunAsUser = %v, want 101", apiSC)
	}
	if apiSC == nil || apiSC.FSGroup == nil || *apiSC.FSGroup != 101 {
		t.Errorf("api FSGroup = %v, want 101", apiSC)
	}

	contentDep := deploySecurityContextComponent(t, pulp, settings.CONTENT)
	contentSC := contentDep.Spec.Template.Spec.SecurityContext
	if contentSC == nil || contentSC.RunAsUser == nil || *contentSC.RunAsUser != 700 {
		t.Errorf("content RunAsUser leaked override or lost default: %v", contentSC)
	}
	if contentSC == nil || contentSC.FSGroup == nil || *contentSC.FSGroup != 700 {
		t.Errorf("content FSGroup leaked override or lost default: %v", contentSC)
	}

	workerDep := deploySecurityContextComponent(t, pulp, settings.WORKER)
	workerSC := workerDep.Spec.Template.Spec.SecurityContext
	if workerSC == nil || workerSC.RunAsUser != nil {
		t.Errorf("worker RunAsUser = %v, want nil (override only set FSGroup)", workerSC)
	}
	if workerSC == nil || workerSC.FSGroup == nil || *workerSC.FSGroup != 2000 {
		t.Errorf("worker FSGroup = %v, want 2000", workerSC)
	}
}

package ocp

import (
	"context"
	"reflect"
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

func deployOCPComponent(t *testing.T, pulp *pulpv1.Pulp, pulpcoreType settings.PulpcoreType) *appsv1.Deployment {
	t.Helper()
	scheme := runtime.NewScheme()
	_ = pulpv1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	resources := controllers.FunctionResources{
		Context: context.TODO(),
		Client:  fake.NewClientBuilder().WithScheme(scheme).WithObjects(pulp).Build(),
		Pulp:    pulp,
		Scheme:  scheme,
		Logger:  zap.New(),
	}

	switch pulpcoreType {
	case settings.API:
		return DeploymentAPIOCP{}.Deploy(resources).(*appsv1.Deployment)
	case settings.CONTENT:
		return DeploymentContentOCP{}.Deploy(resources).(*appsv1.Deployment)
	default:
		return DeploymentWorkerOCP{}.Deploy(resources).(*appsv1.Deployment)
	}
}

// TestOCPPodSecurityContextDefault verifies that, as before this change, OCP
// deployments get an empty SecurityContext when the CR sets no override.
func TestOCPPodSecurityContextDefault(t *testing.T) {
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
			dep := deployOCPComponent(t, pulp, pulpcoreType)
			sc := dep.Spec.Template.Spec.SecurityContext
			if sc == nil || !reflect.DeepEqual(*sc, corev1.PodSecurityContext{}) {
				t.Errorf("SecurityContext = %v, want empty PodSecurityContext{}", sc)
			}
		})
	}
}

// TestOCPPodSecurityContextOverride verifies that an explicit CR override is
// honored on OCP instead of being silently discarded by the empty-context
// default (this was a real bug caught by review before this test existed).
func TestOCPPodSecurityContextOverride(t *testing.T) {
	runAsUser := int64(101)
	fsGroup := int64(101)
	override := &corev1.PodSecurityContext{RunAsUser: &runAsUser, FSGroup: &fsGroup}

	pulp := &pulpv1.Pulp{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pulp", Namespace: "test-namespace"},
		Spec: pulpv1.PulpSpec{
			FileStorageClass: "standard",
			Api:              pulpv1.Api{Replicas: 1, PodSecurityContext: override},
			Content:          pulpv1.Content{Replicas: 1}, // no override: must still get the empty OCP default
			Worker:           pulpv1.Worker{Replicas: 1},
		},
	}

	apiDep := deployOCPComponent(t, pulp, settings.API)
	apiSC := apiDep.Spec.Template.Spec.SecurityContext
	if apiSC == nil || apiSC.RunAsUser == nil || *apiSC.RunAsUser != 101 {
		t.Errorf("api RunAsUser = %v, want 101 (override must survive the OCP defaulting pass)", apiSC)
	}
	if apiSC == nil || apiSC.FSGroup == nil || *apiSC.FSGroup != 101 {
		t.Errorf("api FSGroup = %v, want 101", apiSC)
	}

	contentDep := deployOCPComponent(t, pulp, settings.CONTENT)
	contentSC := contentDep.Spec.Template.Spec.SecurityContext
	if contentSC == nil || !reflect.DeepEqual(*contentSC, corev1.PodSecurityContext{}) {
		t.Errorf("content SecurityContext = %v, want empty PodSecurityContext{} (no override set)", contentSC)
	}
}

package repo_manager

import (
	"context"
	"testing"

	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	"github.com/pulp/pulp-operator/controllers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func deployWeb(t *testing.T, pulp *pulpv1.Pulp) *corev1.PodSecurityContext {
	t.Helper()
	scheme := runtime.NewScheme()
	_ = pulpv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	r := &RepoManagerReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(pulp).Build(),
		Scheme: scheme,
	}
	funcResources := controllers.FunctionResources{
		Context: context.TODO(),
		Client:  r.Client,
		Pulp:    pulp,
		Scheme:  scheme,
		Logger:  zap.New(),
	}

	dep := r.deploymentForPulpWeb(pulp, funcResources)
	return dep.Spec.Template.Spec.SecurityContext
}

// TestPodSecurityContextWebDefault verifies pulp-web keeps the historical
// hardcoded runAsUser/fsGroup 700 when no override is set.
func TestPodSecurityContextWebDefault(t *testing.T) {
	pulp := &pulpv1.Pulp{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pulp", Namespace: "test-namespace"},
		Spec:       pulpv1.PulpSpec{Web: pulpv1.Web{Replicas: 1}},
	}

	sc := deployWeb(t, pulp)
	if sc == nil || sc.RunAsUser == nil || *sc.RunAsUser != 700 {
		t.Errorf("RunAsUser = %v, want 700", sc)
	}
	if sc == nil || sc.FSGroup == nil || *sc.FSGroup != 700 {
		t.Errorf("FSGroup = %v, want 700", sc)
	}
}

// TestPodSecurityContextWebOverride verifies pod_security_context overrides
// the hardcoded default for pulp-web.
func TestPodSecurityContextWebOverride(t *testing.T) {
	runAsUser := int64(101)
	fsGroup := int64(101)
	pulp := &pulpv1.Pulp{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pulp", Namespace: "test-namespace"},
		Spec: pulpv1.PulpSpec{
			Web: pulpv1.Web{
				Replicas:           1,
				PodSecurityContext: &corev1.PodSecurityContext{RunAsUser: &runAsUser, FSGroup: &fsGroup},
			},
		},
	}

	sc := deployWeb(t, pulp)
	if sc == nil || sc.RunAsUser == nil || *sc.RunAsUser != 101 {
		t.Errorf("RunAsUser = %v, want 101", sc)
	}
	if sc == nil || sc.FSGroup == nil || *sc.FSGroup != 101 {
		t.Errorf("FSGroup = %v, want 101", sc)
	}
}

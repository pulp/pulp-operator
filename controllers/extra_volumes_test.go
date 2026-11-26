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

func extraVol(name string) corev1.Volume {
	return corev1.Volume{Name: name, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}
}

func extraMount(name string) corev1.VolumeMount {
	return corev1.VolumeMount{Name: name, MountPath: "/mnt/" + name}
}

// deployComponent runs the real Deployment generator of a pulpcore component
func deployComponent(t *testing.T, pulp *pulpv1.Pulp, pulpcoreType settings.PulpcoreType) *appsv1.Deployment {
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

func newExtraVolumesPulp() *pulpv1.Pulp {
	return &pulpv1.Pulp{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pulp", Namespace: "test-namespace"},
		Spec: pulpv1.PulpSpec{
			FileStorageClass: "standard",
			ImagePullSecrets: []string{"regcred"},
			Api: pulpv1.Api{
				Replicas:           1,
				ServiceAccountName: "api-sa",
				ExtraVolumes:       []corev1.Volume{extraVol("api-vol")},
				ExtraVolumeMounts:  []corev1.VolumeMount{extraMount("api-vol")},
			},
			Worker: pulpv1.Worker{
				Replicas:           1,
				ServiceAccountName: "worker-sa",
				ExtraVolumes:       []corev1.Volume{extraVol("worker-vol"), extraVol("worker-vol2")},
				ExtraVolumeMounts:  []corev1.VolumeMount{extraMount("worker-vol"), extraMount("worker-vol2")},
			},
			// content has no custom settings
			Content: pulpv1.Content{Replicas: 1},
		},
	}
}

func volumeNames(vols []corev1.Volume) []string {
	names := []string{}
	for _, v := range vols {
		names = append(names, v.Name)
	}
	return names
}

func mainContainer(t *testing.T, dep *appsv1.Deployment, name string) corev1.Container {
	t.Helper()
	for _, c := range dep.Spec.Template.Spec.Containers {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("container %q not found in %s", name, dep.Name)
	return corev1.Container{}
}

func TestExtraVolumesGenerator(t *testing.T) {
	tests := []struct {
		pulpcoreType    settings.PulpcoreType
		container       string
		wantVolumes     []string
		wantSA          string
		wantPullSecrets []corev1.LocalObjectReference
	}{
		{settings.API, "api", []string{"api-vol"}, "api-sa", []corev1.LocalObjectReference{{Name: "regcred"}}},
		{settings.WORKER, "worker", []string{"worker-vol", "worker-vol2"}, "worker-sa", []corev1.LocalObjectReference{{Name: "regcred"}}},
		{settings.CONTENT, "content", nil, "test-pulp", nil},
	}

	for _, tt := range tests {
		t.Run(string(tt.pulpcoreType), func(t *testing.T) {
			pulp := newExtraVolumesPulp()
			dep := deployComponent(t, pulp, tt.pulpcoreType)
			podSpec := dep.Spec.Template.Spec

			if podSpec.ServiceAccountName != tt.wantSA {
				t.Errorf("ServiceAccountName = %q, want %q", podSpec.ServiceAccountName, tt.wantSA)
			}
			if len(podSpec.ImagePullSecrets) != len(tt.wantPullSecrets) {
				t.Fatalf("ImagePullSecrets = %v, want %v", podSpec.ImagePullSecrets, tt.wantPullSecrets)
			}
			for i := range tt.wantPullSecrets {
				if podSpec.ImagePullSecrets[i] != tt.wantPullSecrets[i] {
					t.Errorf("ImagePullSecrets[%d] = %v, want %v", i, podSpec.ImagePullSecrets[i], tt.wantPullSecrets[i])
				}
			}

			// operator-managed volumes must be kept and come first
			names := volumeNames(podSpec.Volumes)
			seen := map[string]bool{}
			for _, n := range names {
				if seen[n] {
					t.Errorf("duplicate volume name %q in %v", n, names)
				}
				seen[n] = true
			}
			if !seen[pulp.Name+"-server"] {
				t.Errorf("operator-managed volume %q is missing: %v", pulp.Name+"-server", names)
			}
			if got := names[len(names)-len(tt.wantVolumes):]; len(tt.wantVolumes) > 0 && !equalStrings(got, tt.wantVolumes) {
				t.Errorf("extra volumes should be appended last in order: got %v, want suffix %v", names, tt.wantVolumes)
			}

			// mounts only in the main container, never in init-containers
			main := mainContainer(t, dep, tt.container)
			mounts := map[string]string{}
			for _, m := range main.VolumeMounts {
				mounts[m.Name] = m.MountPath
			}
			if _, found := mounts[pulp.Name+"-server"]; !found {
				t.Errorf("operator-managed mount %q is missing", pulp.Name+"-server")
			}
			for _, name := range tt.wantVolumes {
				if mounts[name] != "/mnt/"+name {
					t.Errorf("mount %q = %q, want %q", name, mounts[name], "/mnt/"+name)
				}
				for _, ic := range podSpec.InitContainers {
					for _, m := range ic.VolumeMounts {
						if m.Name == name {
							t.Errorf("extra mount %q leaked into init-container %q", name, ic.Name)
						}
					}
				}
			}

			// components must not receive other components' extras
			for _, other := range []string{"api-vol", "worker-vol", "worker-vol2"} {
				if contains(tt.wantVolumes, other) {
					continue
				}
				if seen[other] {
					t.Errorf("volume %q from another component leaked into %s", other, tt.pulpcoreType)
				}
				if _, found := mounts[other]; found {
					t.Errorf("mount %q from another component leaked into %s", other, tt.pulpcoreType)
				}
			}
		})
	}
}

// TestExtraVolumesDefaults verifies that nothing changes when the fields are not set
func TestExtraVolumesDefaults(t *testing.T) {
	for _, pulpcoreType := range []settings.PulpcoreType{settings.API, settings.CONTENT, settings.WORKER} {
		t.Run(string(pulpcoreType), func(t *testing.T) {
			pulp := newExtraVolumesPulp()
			pulp.Spec.Api = pulpv1.Api{Replicas: 1}
			pulp.Spec.Content = pulpv1.Content{Replicas: 1}
			pulp.Spec.Worker = pulpv1.Worker{Replicas: 1}

			podSpec := deployComponent(t, pulp, pulpcoreType).Spec.Template.Spec
			if podSpec.ServiceAccountName != settings.PulpServiceAccount(pulp.Name) {
				t.Errorf("ServiceAccountName = %q, want operator-managed one", podSpec.ServiceAccountName)
			}
			if podSpec.ImagePullSecrets != nil {
				t.Errorf("ImagePullSecrets = %v, want nil (they are set in the managed ServiceAccount)", podSpec.ImagePullSecrets)
			}
		})
	}
}

// TestExtraVolumesDoNotMutateSpec verifies that building the Deployment does not change
// the CR (e.g. through slice aliasing)
func TestExtraVolumesDoNotMutateSpec(t *testing.T) {
	pulp := newExtraVolumesPulp()
	// spare capacity makes an unsafe append write into the CR's backing array
	vols := make([]corev1.Volume, 1, 8)
	vols[0] = extraVol("api-vol")
	pulp.Spec.Api.ExtraVolumes = vols
	mounts := make([]corev1.VolumeMount, 1, 8)
	mounts[0] = extraMount("api-vol")
	pulp.Spec.Api.ExtraVolumeMounts = mounts

	deployComponent(t, pulp, settings.API)
	deployComponent(t, pulp, settings.API)

	if got := vols[:2][1]; got.Name != "" {
		t.Errorf("CR volumes backing array was written to: %v", got)
	}
	if got := mounts[:2][1]; got.Name != "" {
		t.Errorf("CR volume mounts backing array was written to: %v", got)
	}
	if len(pulp.Spec.Api.ExtraVolumes) != 1 || len(pulp.Spec.Api.ExtraVolumeMounts) != 1 {
		t.Errorf("CR spec was modified")
	}
}

// TestExtraVolumesHashLabel verifies that changing the fields rolls the Deployment and
// that an unchanged CR does not
func TestExtraVolumesHashLabel(t *testing.T) {
	hash := func(pulp *pulpv1.Pulp) string {
		return deployComponent(t, pulp, settings.WORKER).Labels[OperatorHashLabel]
	}

	base := hash(newExtraVolumesPulp())
	if base == "" {
		t.Fatal("hash label not set")
	}
	if again := hash(newExtraVolumesPulp()); again != base {
		t.Errorf("same spec produced different hashes: %s != %s", base, again)
	}

	changes := map[string]func(p *pulpv1.Pulp){
		"service account": func(p *pulpv1.Pulp) { p.Spec.Worker.ServiceAccountName = "other-sa" },
		"volume":          func(p *pulpv1.Pulp) { p.Spec.Worker.ExtraVolumes[0] = extraVol("renamed") },
		"mount path": func(p *pulpv1.Pulp) {
			p.Spec.Worker.ExtraVolumeMounts[0].MountPath = "/other"
		},
		"mount read-only": func(p *pulpv1.Pulp) { p.Spec.Worker.ExtraVolumeMounts[0].ReadOnly = true },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			pulp := newExtraVolumesPulp()
			change(pulp)
			if got := hash(pulp); got == base {
				t.Errorf("changing %s did not change the hash", name)
			}
		})
	}

	// a change in another component must not roll this one
	pulp := newExtraVolumesPulp()
	pulp.Spec.Api.ServiceAccountName = "other-api-sa"
	if got := hash(pulp); got != base {
		t.Errorf("changing the api service account changed the worker hash")
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

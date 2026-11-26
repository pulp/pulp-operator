package repo_manager_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CRD schema validation of extra_volumes, extra_volume_mounts and service_account_name
var _ = Describe("Pulp CRD validation of extra volumes", func() {
	ctx := context.Background()
	components := []string{"api", "content", "worker"}

	newPulp := func(name string, spec map[string]any) *unstructured.Unstructured {
		u := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": pulpv1.GroupVersion.String(),
			"kind":       "Pulp",
			"metadata":   map[string]any{"name": name, "namespace": "default"},
			"spec":       spec,
		}}
		return u
	}

	volume := func(name string) map[string]any {
		return map[string]any{
			"name": name,
			"csi": map[string]any{
				"driver":           "secrets-store.csi.k8s.io",
				"readOnly":         true,
				"volumeAttributes": map[string]any{"secretProviderClass": "my-secrets"},
			},
		}
	}

	create := func(name string, spec map[string]any) error {
		u := newPulp(name, spec)
		err := k8sClient.Create(ctx, u)
		if err == nil {
			DeferCleanup(func() { _ = k8sClient.Delete(ctx, u) })
		}
		return err
	}

	for _, component := range components {
		component := component

		It("accepts and preserves the fields in "+component, func() {
			name := "ev-valid-" + component
			Expect(create(name, map[string]any{
				component: map[string]any{
					"service_account_name": "custom-sa",
					"extra_volumes": []any{
						volume("secrets-store"),
						map[string]any{"name": "scripts", "secret": map[string]any{"secretName": "my-scripts", "defaultMode": int64(493)}},
					},
					"extra_volume_mounts": []any{
						map[string]any{"name": "secrets-store", "mountPath": "/mnt/secrets", "readOnly": true},
						map[string]any{"name": "scripts", "mountPath": "/opt/pulp-scripts", "subPath": "run.sh"},
					},
				},
			})).To(Succeed())

			// the fields must not be pruned by the API server
			got := &unstructured.Unstructured{}
			got.SetGroupVersionKind(pulpv1.GroupVersion.WithKind("Pulp"))
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: "default"}, got)).To(Succeed())

			sa, _, _ := unstructured.NestedString(got.Object, "spec", component, "service_account_name")
			Expect(sa).To(Equal("custom-sa"))

			vols, _, _ := unstructured.NestedSlice(got.Object, "spec", component, "extra_volumes")
			Expect(vols).To(HaveLen(2))
			driver, _, _ := unstructured.NestedString(vols[0].(map[string]any), "csi", "driver")
			Expect(driver).To(Equal("secrets-store.csi.k8s.io"))
			mode, _, _ := unstructured.NestedInt64(vols[1].(map[string]any), "secret", "defaultMode")
			Expect(mode).To(Equal(int64(493)))

			mounts, _, _ := unstructured.NestedSlice(got.Object, "spec", component, "extra_volume_mounts")
			Expect(mounts).To(HaveLen(2))
			Expect(mounts[1].(map[string]any)["subPath"]).To(Equal("run.sh"))

			// and must round-trip into the typed API
			typed := &pulpv1.Pulp{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: "default"}, typed)).To(Succeed())
			switch component {
			case "api":
				Expect(typed.Spec.Api.ExtraVolumes).To(HaveLen(2))
				Expect(typed.Spec.Api.ServiceAccountName).To(Equal("custom-sa"))
			case "content":
				Expect(typed.Spec.Content.ExtraVolumes).To(HaveLen(2))
				Expect(typed.Spec.Content.ServiceAccountName).To(Equal("custom-sa"))
			case "worker":
				Expect(typed.Spec.Worker.ExtraVolumes).To(HaveLen(2))
				Expect(typed.Spec.Worker.ServiceAccountName).To(Equal("custom-sa"))
			}
		})

		DescribeTable("rejects invalid values in "+component,
			func(field string, value any, wantErr string) {
				err := create("ev-invalid-"+component, map[string]any{
					component: map[string]any{field: value},
				})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(wantErr))
			},
			Entry("volume without name", "extra_volumes", []any{map[string]any{"emptyDir": map[string]any{}}}, "name"),
			Entry("mount without mountPath", "extra_volume_mounts", []any{map[string]any{"name": "scripts"}}, "mountPath"),
			Entry("mount without name", "extra_volume_mounts", []any{map[string]any{"mountPath": "/mnt"}}, "name"),
			Entry("AWX-style string instead of a list of volumes", "extra_volumes", "- name: scripts\n  emptyDir: {}\n", "extra_volumes"),
			Entry("string instead of a list of mounts", "extra_volume_mounts", "- name: scripts\n", "extra_volume_mounts"),
			Entry("non-string service account name", "service_account_name", int64(42), "service_account_name"),
			Entry("readOnly with the wrong type", "extra_volume_mounts", []any{map[string]any{"name": "a", "mountPath": "/a", "readOnly": "yes"}}, "readOnly"),
		)
	}

	It("keeps the fields optional", func() {
		Expect(create("ev-empty", map[string]any{
			"api":     map[string]any{"replicas": int64(1)},
			"content": map[string]any{"replicas": int64(1)},
			"worker":  map[string]any{"replicas": int64(1)},
		})).To(Succeed())

		got := &pulpv1.Pulp{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: "ev-empty", Namespace: "default"}, got)).To(Succeed())
		Expect(got.Spec.Api.ExtraVolumes).To(BeEmpty())
		Expect(got.Spec.Worker.ExtraVolumeMounts).To(BeEmpty())
		Expect(got.Spec.Content.ServiceAccountName).To(BeEmpty())
	})
})

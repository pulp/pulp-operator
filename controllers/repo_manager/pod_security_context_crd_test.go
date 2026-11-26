package repo_manager_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

// CRD schema validation of pod_security_context
var _ = Describe("Pulp CRD validation of pod_security_context", func() {
	ctx := context.Background()
	components := []string{"api", "content", "worker", "web"}

	newPulp := func(name string, spec map[string]any) *unstructured.Unstructured {
		return &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": pulpv1.GroupVersion.String(),
			"kind":       "Pulp",
			"metadata":   map[string]any{"name": name, "namespace": "default"},
			"spec":       spec,
		}}
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

		It("accepts and preserves pod_security_context in "+component, func() {
			name := "psc-valid-" + component
			Expect(create(name, map[string]any{
				component: map[string]any{
					"pod_security_context": map[string]any{
						"runAsUser":  int64(101),
						"fsGroup":    int64(101),
						"runAsGroup": int64(101),
					},
				},
			})).To(Succeed())

			// the field must not be pruned by the API server
			got := &unstructured.Unstructured{}
			got.SetGroupVersionKind(pulpv1.GroupVersion.WithKind("Pulp"))
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: "default"}, got)).To(Succeed())

			runAsUser, _, _ := unstructured.NestedInt64(got.Object, "spec", component, "pod_security_context", "runAsUser")
			Expect(runAsUser).To(Equal(int64(101)))

			fsGroup, _, _ := unstructured.NestedInt64(got.Object, "spec", component, "pod_security_context", "fsGroup")
			Expect(fsGroup).To(Equal(int64(101)))

			runAsGroup, _, _ := unstructured.NestedInt64(got.Object, "spec", component, "pod_security_context", "runAsGroup")
			Expect(runAsGroup).To(Equal(int64(101)))
		})
	}

	It("accepts a CR with no pod_security_context override (default behavior unchanged)", func() {
		name := "psc-unset"
		Expect(create(name, map[string]any{
			"api": map[string]any{"replicas": int64(1)},
		})).To(Succeed())

		got := &unstructured.Unstructured{}
		got.SetGroupVersionKind(pulpv1.GroupVersion.WithKind("Pulp"))
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: "default"}, got)).To(Succeed())

		_, found, _ := unstructured.NestedMap(got.Object, "spec", "api", "pod_security_context")
		Expect(found).To(BeFalse())
	})
})

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
	"path/filepath"
	"reflect"
	"testing"

	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	corev1 "k8s.io/api/core/v1"
)

// isolateFromClusters keeps commonJob's OpenShift probe away from any real kubeconfig.
func isolateFromClusters(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("KUBECONFIG", filepath.Join(dir, "missing"))
}

func TestCommonJobScheduling(t *testing.T) {
	isolateFromClusters(t)

	scheduling := pulpv1.PulpJob{
		NodeSelector: map[string]string{"kubernetes.io/os": "linux"},
		Tolerations: []corev1.Toleration{{
			Key:      "dedicated",
			Operator: corev1.TolerationOpEqual,
			Value:    "pulp",
			Effect:   corev1.TaintEffectNoSchedule,
		}},
		Affinity: &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{{
						MatchExpressions: []corev1.NodeSelectorRequirement{{
							Key:      "topology.kubernetes.io/zone",
							Operator: corev1.NodeSelectorOpIn,
							Values:   []string{"zone-a"},
						}},
					}},
				},
			},
		},
	}

	job := commonJob(pulpJobConfig{name: "test-job-", namespace: "test", scheduling: scheduling})
	spec := job.Spec.Template.Spec

	if !reflect.DeepEqual(spec.NodeSelector, scheduling.NodeSelector) {
		t.Errorf("nodeSelector = %v, want %v", spec.NodeSelector, scheduling.NodeSelector)
	}
	if !reflect.DeepEqual(spec.Tolerations, scheduling.Tolerations) {
		t.Errorf("tolerations = %v, want %v", spec.Tolerations, scheduling.Tolerations)
	}
	if !reflect.DeepEqual(spec.Affinity, scheduling.Affinity) {
		t.Errorf("affinity = %v, want %v", spec.Affinity, scheduling.Affinity)
	}
}

func TestCommonJobWithoutScheduling(t *testing.T) {
	isolateFromClusters(t)

	spec := commonJob(pulpJobConfig{name: "test-job-", namespace: "test"}).Spec.Template.Spec

	if spec.NodeSelector != nil || spec.Tolerations != nil || spec.Affinity != nil {
		t.Errorf("expected no scheduling constraints, got nodeSelector=%v tolerations=%v affinity=%v",
			spec.NodeSelector, spec.Tolerations, spec.Affinity)
	}
}

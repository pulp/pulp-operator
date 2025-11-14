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
	"reflect"

	"github.com/go-logr/logr"
	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	"github.com/pulp/pulp-operator/controllers/settings"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

// hpaController manages HorizontalPodAutoscaler resources for Pulp components
func (r *RepoManagerReconciler) hpaController(ctx context.Context, pulp *pulpv1.Pulp, log logr.Logger) (ctrl.Result, error) {
	// Handle HPA for each component
	components := []settings.PulpcoreType{settings.API, settings.CONTENT, settings.WORKER}

	for _, component := range components {
		if err := r.reconcileHPA(ctx, pulp, component, log); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Handle Web component HPA if web is deployed
	if !isRoute(pulp) && !isIngress(pulp) {
		if err := r.reconcileHPA(ctx, pulp, settings.WEB, log); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// reconcileHPA creates, updates, or deletes HPA for a specific component
func (r *RepoManagerReconciler) reconcileHPA(ctx context.Context, pulp *pulpv1.Pulp, pulpcoreType settings.PulpcoreType, log logr.Logger) error {
	hpaConfig := getHPAConfig(pulp, pulpcoreType)
	hpaName := pulpcoreType.DeploymentName(pulp.Name)

	// Check if HPA exists
	foundHPA := &autoscalingv2.HorizontalPodAutoscaler{}
	err := r.Get(ctx, types.NamespacedName{Name: hpaName, Namespace: pulp.Namespace}, foundHPA)

	// If HPA is disabled or not configured, delete existing HPA if present
	if hpaConfig == nil || !hpaConfig.Enabled {
		if err == nil {
			log.Info("Deleting HPA", "Component", pulpcoreType, "HPA.Name", hpaName)
			if err := r.Delete(ctx, foundHPA); err != nil {
				log.Error(err, "Failed to delete HPA", "Component", pulpcoreType)
				return err
			}
		}
		return nil
	}

	// Build desired HPA
	desiredHPA := buildHPA(pulp, pulpcoreType, hpaConfig)

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(pulp, desiredHPA, r.Scheme)

	// If HPA doesn't exist, create it
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating HPA", "Component", pulpcoreType, "HPA.Name", hpaName)
		if err := r.Create(ctx, desiredHPA); err != nil {
			log.Error(err, "Failed to create HPA", "Component", pulpcoreType)
			return err
		}
		return nil
	} else if err != nil {
		log.Error(err, "Failed to get HPA", "Component", pulpcoreType)
		return err
	}

	// HPA exists, check if update is needed
	if !reflect.DeepEqual(foundHPA.Spec, desiredHPA.Spec) {
		log.Info("Updating HPA", "Component", pulpcoreType, "HPA.Name", hpaName)
		foundHPA.Spec = desiredHPA.Spec
		if err := r.Update(ctx, foundHPA); err != nil {
			log.Error(err, "Failed to update HPA", "Component", pulpcoreType)
			return err
		}
	}

	return nil
}

// getHPAConfig retrieves HPA configuration for a specific component
func getHPAConfig(pulp *pulpv1.Pulp, pulpcoreType settings.PulpcoreType) *pulpv1.HPA {
	specField := reflect.ValueOf(pulp.Spec).FieldByName(string(pulpcoreType))
	if !specField.IsValid() {
		return nil
	}

	hpaField := specField.FieldByName("HPA")
	if !hpaField.IsValid() || hpaField.IsNil() {
		return nil
	}

	return hpaField.Interface().(*pulpv1.HPA)
}

// buildHPA constructs a HorizontalPodAutoscaler object
func buildHPA(pulp *pulpv1.Pulp, pulpcoreType settings.PulpcoreType, hpaConfig *pulpv1.HPA) *autoscalingv2.HorizontalPodAutoscaler {
	hpaName := pulpcoreType.DeploymentName(pulp.Name)
	labels := settings.PulpcoreLabels(*pulp, string(pulpcoreType))

	// Set default min replicas if not specified
	minReplicas := hpaConfig.MinReplicas
	if minReplicas == nil {
		defaultMin := int32(1)
		minReplicas = &defaultMin
	}

	// Build metrics
	metrics := []autoscalingv2.MetricSpec{}

	// Add CPU metric if specified
	if hpaConfig.TargetCPUUtilizationPercentage != nil {
		metrics = append(metrics, autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceCPU,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: hpaConfig.TargetCPUUtilizationPercentage,
				},
			},
		})
	}

	// Add Memory metric if specified
	if hpaConfig.TargetMemoryUtilizationPercentage != nil {
		metrics = append(metrics, autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceMemory,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: hpaConfig.TargetMemoryUtilizationPercentage,
				},
			},
		})
	}

	// If no metrics specified, default to 50% CPU
	if len(metrics) == 0 {
		defaultCPU := int32(50)
		metrics = append(metrics, autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceCPU,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: &defaultCPU,
				},
			},
		})
	}

	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hpaName,
			Namespace: pulp.Namespace,
			Labels:    labels,
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       hpaName,
			},
			MinReplicas: minReplicas,
			MaxReplicas: hpaConfig.MaxReplicas,
			Metrics:     metrics,
		},
	}

	return hpa
}

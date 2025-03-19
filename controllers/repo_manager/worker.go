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

	"github.com/go-logr/logr"
	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	"github.com/pulp/pulp-operator/controllers"
	"github.com/pulp/pulp-operator/controllers/settings"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *RepoManagerReconciler) pulpWorkerController(ctx context.Context, pulp *pulpv1.Pulp, log logr.Logger) (ctrl.Result, error) {

	// conditionType is used to update .status.conditions with the current resource state
	conditionType := "Pulp-Worker-Ready"

	// temporary workaround that creates a configmap to be used as a readiness probe when ipv6 is disabled (pending oci-image update)
	if requeue, err := r.createProbeConfigMap(ctx, pulp, conditionType); err != nil || requeue != nil {
		return *requeue, err
	}

	funcResources := controllers.FunctionResources{Context: ctx, Client: r.Client, Pulp: pulp, Scheme: r.Scheme, Logger: log}

	// define the k8s Deployment function based on k8s distribution and deployment type
	deploymentForPulpWorker := initDeployment(WORKER_DEPLOYMENT).Deploy
	deploymentName := settings.WORKER.DeploymentName(pulp.Name)

	// Create Worker Deployment
	if requeue, err := r.createPulpResource(ResourceDefinition{ctx, &appsv1.Deployment{}, deploymentName, "Worker", conditionType, pulp}, deploymentForPulpWorker); err != nil || requeue {
		return ctrl.Result{Requeue: requeue}, err
	}

	// Reconcile Deployment
	found := &appsv1.Deployment{}
	r.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: pulp.Namespace}, found)
	expected := deploymentForPulpWorker(funcResources)
	if requeue, err := controllers.ReconcileObject(funcResources, expected, found, conditionType, controllers.PulpDeployment{}); err != nil || requeue {
		return ctrl.Result{Requeue: requeue}, err
	}

	// we should only update the status when Worker-Ready==false
	if v1.IsStatusConditionFalse(pulp.Status.Conditions, conditionType) {
		controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionTrue, conditionType, "WorkerTasksFinished", "All Worker tasks ran successfully")
		r.recorder.Event(pulp, corev1.EventTypeNormal, "WorkerReady", "All Worker tasks ran successfully")
	}
	return ctrl.Result{}, nil
}

// TODO: the ipv6 incompatibility should be handled by oci-image.
// Remove this function after updating the image.
func (r *RepoManagerReconciler) createProbeConfigMap(ctx context.Context, pulp *pulpv1.Pulp, conditionType string) (*ctrl.Result, error) {

	if !controllers.Ipv6Disabled(*pulp) {
		return nil, nil
	}

	configMapName := settings.PulpWorkerProbe(pulp.Name)
	resourceDefinition := ResourceDefinition{
		Context:       ctx,
		Type:          &corev1.ConfigMap{},
		Name:          configMapName,
		Alias:         "PulpWorkerProbe",
		ConditionType: conditionType,
		Pulp:          pulp}

	// create the configmap
	requeue, err := r.createPulpResource(resourceDefinition, postgresConnectionConfigMap)
	if err != nil {
		return nil, err
	} else if requeue {
		return &ctrl.Result{Requeue: true}, nil
	}

	// Ensure the configmap data is as expected
	funcResources := controllers.FunctionResources{Context: ctx, Client: r.Client, Pulp: pulp, Scheme: r.Scheme, Logger: r.RawLogger}
	configMap := &corev1.ConfigMap{}
	r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: pulp.Namespace}, configMap)
	expectedCM := postgresConnectionConfigMap(funcResources)
	if requeue, err := controllers.ReconcileObject(funcResources, expectedCM, configMap, conditionType, controllers.PulpConfigMap{}); err != nil || requeue {
		return &ctrl.Result{Requeue: requeue}, err
	}

	return nil, nil
}

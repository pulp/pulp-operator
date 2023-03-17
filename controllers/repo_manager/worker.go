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
	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	"github.com/pulp/pulp-operator/controllers"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *RepoManagerReconciler) pulpWorkerController(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp, log logr.Logger) (ctrl.Result, error) {

	// conditionType is used to update .status.conditions with the current resource state
	conditionType := cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Worker-Ready"
	funcResources := controllers.FunctionResources{Context: ctx, Client: r.Client, Pulp: pulp, Scheme: r.Scheme, Logger: log}

	// define the k8s Deployment function based on k8s distribution and deployment type
	deploymentForPulpWorker := initDeployment(WORKER_DEPLOYMENT).Deploy

	// Create Worker Deployment
	if requeue, err := r.createPulpResource(ResourceDefinition{ctx, &appsv1.Deployment{}, pulp.Name + "-worker", "Worker", conditionType, pulp}, deploymentForPulpWorker); err != nil || requeue {
		return ctrl.Result{Requeue: requeue}, err
	}

	// Reconcile Deployment
	found := &appsv1.Deployment{}
	r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-worker", Namespace: pulp.Namespace}, found)
	expected := deploymentForPulpWorker(funcResources)
	if requeue, err := controllers.ReconcileObject(funcResources, expected, found, conditionType); err != nil || requeue {
		return ctrl.Result{Requeue: requeue}, err
	}

	// we should only update the status when Worker-Ready==false
	if v1.IsStatusConditionFalse(pulp.Status.Conditions, conditionType) {
		controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionTrue, conditionType, "WorkerTasksFinished", "All Worker tasks ran successfully")
		r.recorder.Event(pulp, corev1.EventTypeNormal, "WorkerReady", "All Worker tasks ran successfully")
	}
	return ctrl.Result{}, nil
}

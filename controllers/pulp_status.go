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

package controllers

import (
	"context"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
	"github.com/go-logr/logr"
)

func (r *PulpReconciler) pulpStatus(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, log logr.Logger) (ctrl.Result, error) {

	apiDeployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-api", Namespace: pulp.Namespace}, apiDeployment)
	if err == nil {
		if apiDeployment.Status.ReadyReplicas != apiDeployment.Status.Replicas {
			log.Info("Pulp api not ready yet ...")
			return ctrl.Result{Requeue: true, RequeueAfter: 30 * time.Second}, nil
		} else {
			r.updateStatus(ctx, pulp, metav1.ConditionTrue, pulp.Spec.DeploymentType+"-API-Ready", "ApiTasksFinished", "All API tasks ran successfully")
		}
	} else {
		log.Error(err, "Failed to get Pulp API Deployment")
		return ctrl.Result{Requeue: true, RequeueAfter: 10 * time.Second}, nil
	}

	webDeployment := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-web", Namespace: pulp.Namespace}, webDeployment)
	if err == nil {
		if webDeployment.Status.ReadyReplicas != webDeployment.Status.Replicas {
			log.Info("Pulp web not ready yet ...")
			return ctrl.Result{Requeue: true, RequeueAfter: 30 * time.Second}, nil
		} else {
			r.updateStatus(ctx, pulp, metav1.ConditionTrue, pulp.Spec.DeploymentType+"-Web-Ready", "WebTasksFinished", "All Web tasks ran successfully")
		}
	} else {
		log.Error(err, "Failed to get Pulp Web Deployment")
		return ctrl.Result{Requeue: true, RequeueAfter: 10 * time.Second}, nil
	}

	if v1.IsStatusConditionFalse(pulp.Status.Conditions, cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType)+"-Operator-Finished-Execution") {
		v1.SetStatusCondition(&pulp.Status.Conditions, metav1.Condition{
			Type:               cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Operator-Finished-Execution",
			Status:             metav1.ConditionTrue,
			Reason:             "OperatorFinishedExecution",
			LastTransitionTime: metav1.Now(),
			Message:            "All tasks ran successfully",
		})
		r.Status().Update(ctx, pulp)
		log.Info(pulp.Spec.DeploymentType + " operator finished execution ...")
	}
	return ctrl.Result{}, nil
}

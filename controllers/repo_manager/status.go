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
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/go-logr/logr"
	repomanagerv1alpha1 "github.com/pulp/pulp-operator/api/v1alpha1"
	"github.com/pulp/pulp-operator/controllers"
)

// pulpStatus will cheeck the READY state of the pods before considering the component status as ready
func (r *RepoManagerReconciler) pulpStatus(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, log logr.Logger) (ctrl.Result, error) {

	// This is a very ugly workaround to "fix" a possible race condition issue.
	// During a reconciliation task we call the pulpStatus method to update the .status.conditions field.
	// Without the sleep, when we do the isDeploymentReady check, the deployment can still be with the
	// "old" state. This 0,2 seconds seems to be enough to delay the check and reflect the real state to
	// the controller.
	time.Sleep(time.Millisecond * 200)

	// check if Content pods are READY
	contentDeployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-content", Namespace: pulp.Namespace}, contentDeployment); err == nil {
		contentConditionType := cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Content-Ready"
		if !isDeploymentReady(contentDeployment) {
			log.Info(pulp.Spec.DeploymentType + " content not ready yet ...")
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, contentConditionType, "UpdatingContentDeployment", "Content deployment not ready yet")
			return ctrl.Result{Requeue: true, RequeueAfter: 30 * time.Second}, nil
		} else if v1.IsStatusConditionFalse(pulp.Status.Conditions, contentConditionType) {
			r.updateStatus(ctx, pulp, metav1.ConditionTrue, contentConditionType, "ContentTasksFinished", "All Content tasks ran successfully")
			r.recorder.Event(pulp, corev1.EventTypeNormal, "ContentReady", "All Content tasks ran successfully")
		}
	} else {
		log.Error(err, "Failed to get Pulp Content Deployment")
		return ctrl.Result{Requeue: true, RequeueAfter: 10 * time.Second}, nil
	}

	// check if API pods are READY
	apiDeployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-api", Namespace: pulp.Namespace}, apiDeployment); err == nil {
		apiConditionType := cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-API-Ready"
		if !isDeploymentReady(apiDeployment) {
			log.Info(pulp.Spec.DeploymentType + " api not ready yet ...")
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, apiConditionType, "UpdatingAPIDeployment", "API deployment not ready yet")
			return ctrl.Result{Requeue: true, RequeueAfter: 30 * time.Second}, nil
		} else if v1.IsStatusConditionFalse(pulp.Status.Conditions, apiConditionType) {
			r.updateStatus(ctx, pulp, metav1.ConditionTrue, apiConditionType, "ApiTasksFinished", "All API tasks ran successfully")
			r.recorder.Event(pulp, corev1.EventTypeNormal, "APIReady", "All API tasks ran successfully")
		}
	} else {
		log.Error(err, "Failed to get Pulp API Deployment")
		return ctrl.Result{Requeue: true, RequeueAfter: 10 * time.Second}, nil
	}

	// check web pods are READY
	isNginxIngress := strings.ToLower(pulp.Spec.IngressType) == "ingress" && controllers.IsNginxIngressSupported(r)
	if strings.ToLower(pulp.Spec.IngressType) != "route" && !isNginxIngress {
		webConditionType := cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Web-Ready"
		webDeployment := &appsv1.Deployment{}
		if err := r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-web", Namespace: pulp.Namespace}, webDeployment); err == nil {
			if !isDeploymentReady(webDeployment) {
				log.Info(pulp.Spec.DeploymentType + " web not ready yet ...")
				r.updateStatus(ctx, pulp, metav1.ConditionFalse, webConditionType, "UpdatingWebDeployment", "Web deployment not ready yet")
				return ctrl.Result{Requeue: true, RequeueAfter: 30 * time.Second}, nil
			} else if v1.IsStatusConditionFalse(pulp.Status.Conditions, webConditionType) {
				r.updateStatus(ctx, pulp, metav1.ConditionTrue, webConditionType, "WebTasksFinished", "All Web tasks ran successfully")
				r.recorder.Event(pulp, corev1.EventTypeNormal, "WebReady", "All Web tasks ran successfully")
			}
		} else {
			log.Error(err, "Failed to get Pulp Web Deployment")
			return ctrl.Result{Requeue: true, RequeueAfter: 10 * time.Second}, nil
		}
	}

	// if we get into here it means that all components are READY, so operator finished its execution
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

	/*
		[TODO] refactor the following conditionals to avoid repetitive code
		.status fields that are used by reconcile logic
	*/

	// we will only set .status.deployment_type in the first execution (len==0)
	if len(pulp.Status.DeploymentType) == 0 {
		pulp.Status.DeploymentType = pulp.Spec.DeploymentType
		r.Status().Update(ctx, pulp)
	}

	// we will only set .status.object_storage_azure_secret in the first execution (len==0)
	// and if .spec.object_storage_azure_secret is defined
	if len(pulp.Status.ObjectStorageAzureSecret) == 0 && len(pulp.Spec.ObjectStorageAzureSecret) > 0 {
		pulp.Status.ObjectStorageAzureSecret = pulp.Spec.ObjectStorageAzureSecret
		r.Status().Update(ctx, pulp)
	}

	// we will only set .status.object_storage_s3_secret in the first execution (len==0)
	// and if .spec.object_storage_s3_secret is defined
	if len(pulp.Status.ObjectStorageS3Secret) == 0 && len(pulp.Spec.ObjectStorageS3Secret) > 0 {
		pulp.Status.ObjectStorageS3Secret = pulp.Spec.ObjectStorageS3Secret
		r.Status().Update(ctx, pulp)
	}

	// we will only set .status.db_fields_encryption_secret in the first execution (len==0)
	// and we'll set with the value from pulp.spec.db_fields_encryption_secret if defined
	if len(pulp.Status.DBFieldsEncryptionSecret) == 0 && len(pulp.Spec.DBFieldsEncryptionSecret) > 0 {
		pulp.Status.DBFieldsEncryptionSecret = pulp.Spec.DBFieldsEncryptionSecret
		r.Status().Update(ctx, pulp)

		// if pulp.spec.db_fields_encryption_secret is not defined we should set .status with the default value
	} else if len(pulp.Status.DBFieldsEncryptionSecret) == 0 && len(pulp.Spec.DBFieldsEncryptionSecret) == 0 {
		pulp.Status.DBFieldsEncryptionSecret = pulp.Name + "-db-fields-encryption"
		r.Status().Update(ctx, pulp)
	}

	// if there is no .status.ingress_type defined yet, we'll populate it with the current value
	// this field can be modified (by other functions/methods) if .spec.ingress_type is modified
	if len(pulp.Status.IngressType) == 0 {
		pulp.Status.IngressType = pulp.Spec.IngressType
		r.Status().Update(ctx, pulp)
	}

	// we will only set .status.container_token_secret in the first execution (len==0)
	// and we'll set with the value from pulp.spec.container_token_secret if defined
	if len(pulp.Status.ContainerTokenSecret) == 0 && len(pulp.Spec.ContainerTokenSecret) > 0 {
		pulp.Status.ContainerTokenSecret = pulp.Spec.ContainerTokenSecret
		r.Status().Update(ctx, pulp)

		// if pulp.spec.container_token_secret is not defined we should set .status with the
		// secret created by the operator
	} else if len(pulp.Status.ContainerTokenSecret) == 0 && len(pulp.Spec.ContainerTokenSecret) == 0 {
		pulp.Status.ContainerTokenSecret = pulp.Name + "-container-auth"
		r.Status().Update(ctx, pulp)
	}

	// we will only set .status.admin_password_secret in the first execution (len==0)
	// and we'll set with the value from pulp.spec.admin_password_secret if defined
	if len(pulp.Status.AdminPasswordSecret) == 0 && len(pulp.Spec.AdminPasswordSecret) > 0 {
		pulp.Status.AdminPasswordSecret = pulp.Spec.AdminPasswordSecret
		r.Status().Update(ctx, pulp)

		// if pulp.spec.admin_password_secret is not defined we should set .status with the default value
	} else if len(pulp.Status.AdminPasswordSecret) == 0 && len(pulp.Spec.AdminPasswordSecret) == 0 {
		pulp.Status.AdminPasswordSecret = pulp.Name + "-admin-password"
		r.Status().Update(ctx, pulp)
	}

	// we will only set .status.external_cache_secret in the first execution (len==0)
	// and if .spec.external_cache_secret is defined
	if len(pulp.Status.ExternalCacheSecret) == 0 && len(pulp.Spec.Cache.ExternalCacheSecret) > 0 {
		pulp.Status.ExternalCacheSecret = pulp.Spec.Cache.ExternalCacheSecret
		r.Status().Update(ctx, pulp)
	}

	return ctrl.Result{}, nil
}

// isDeploymentReady returns true if there is no unavailable Replicas for the deployment
func isDeploymentReady(deployment *appsv1.Deployment) bool {

	return deployment.Status.UnavailableReplicas <= 0
}

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
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/go-logr/logr"
	repomanagerv1alpha1 "github.com/pulp/pulp-operator/api/v1alpha1"
)

// pulpResource contains the fields to update the .status.conditions from pulp instance
type pulpResource struct {
	Type          string
	Name          string
	ConditionType string
}

// pulpStatus will cheeck the READY state of the pods before considering the component status as ready
func (r *RepoManagerReconciler) pulpStatus(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, log logr.Logger) (ctrl.Result, error) {

	// This is a very ugly workaround to "fix" a possible race condition issue.
	// During a reconciliation task we call the pulpStatus method to update the .status.conditions field.
	// Without the sleep, when we do the isDeploymentReady check, the deployment can still be with the
	// "old" state. This 0,2 seconds seems to be enough to delay the check and reflect the real state to
	// the controller.
	time.Sleep(time.Millisecond * 200)
	pulpResources := []pulpResource{
		{
			Type:          "content",
			Name:          pulp.Name + "-content",
			ConditionType: cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Content-Ready",
		},
		{
			Type:          "api",
			Name:          pulp.Name + "-api",
			ConditionType: cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-API-Ready",
		},
		{
			Type:          "worker",
			Name:          pulp.Name + "-worker",
			ConditionType: cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Worker-Ready",
		},
		{
			Type:          "web",
			Name:          pulp.Name + "-web",
			ConditionType: cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Web-Ready",
		},
	}

	// each pulpcore status resource (content,worker,api) will be checked in a different go-routine to avoid
	// an issue with one of the status resource not getting updated until the previous one finishes
	for _, resource := range pulpResources {

		// if route or ingress we should do nothing
		if resource.Type == "web" {
			if strings.ToLower(pulp.Spec.IngressType) == "route" || r.isNginxIngress(pulp) {
				continue
			}
			if strings.ToLower(pulp.Spec.IngressType) == "ingress" {
				currentIngress := &netv1.Ingress{}
				r.Get(ctx, types.NamespacedName{Name: pulp.Name, Namespace: pulp.Namespace}, currentIngress)
				if currentIngress.Annotations["web"] == "false" {
					continue
				}
			}
		}

		go func(resource pulpResource) {
			deployment := &appsv1.Deployment{}
			typeCapitalized := cases.Title(language.English, cases.Compact).String(resource.Type)
			if err := r.Get(ctx, types.NamespacedName{Name: resource.Name, Namespace: pulp.Namespace}, deployment); err == nil {
				if !isDeploymentReady(deployment) {
					log.Info(pulp.Spec.DeploymentType + " " + resource.Type + " not ready yet ...")
					r.updateStatus(ctx, pulp, metav1.ConditionFalse, resource.ConditionType, "Updating"+typeCapitalized+"Deployment", typeCapitalized+" deployment not ready yet")
				} else if v1.IsStatusConditionFalse(pulp.Status.Conditions, resource.ConditionType) {
					r.updateStatus(ctx, pulp, metav1.ConditionTrue, resource.ConditionType, typeCapitalized+"TasksFinished", "All "+typeCapitalized+" tasks ran successfully")
					r.recorder.Event(pulp, corev1.EventTypeNormal, typeCapitalized+"Ready", "All "+typeCapitalized+" tasks ran successfully")
				}
			} else {
				log.Error(err, "Failed to get Pulp "+typeCapitalized+" Deployment")
			}
		}(resource)
	}

	// requeue until all deployments get READY
	r.Get(ctx, types.NamespacedName{Name: pulp.Name, Namespace: pulp.Namespace}, pulp)
	for _, resource := range pulpResources {
		if v1.IsStatusConditionFalse(pulp.Status.Conditions, resource.ConditionType) {
			return ctrl.Result{RequeueAfter: time.Second * 10}, nil
		}
	}

	// if we get into here it means that all components are READY, so operator finished its execution
	if v1.IsStatusConditionFalse(pulp.Status.Conditions, cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType)+"-Operator-Finished-Execution") {
		r.Get(ctx, types.NamespacedName{Name: pulp.Name, Namespace: pulp.Namespace}, pulp)
		v1.SetStatusCondition(&pulp.Status.Conditions, metav1.Condition{
			Type:               cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Operator-Finished-Execution",
			Status:             metav1.ConditionTrue,
			Reason:             "OperatorFinishedExecution",
			LastTransitionTime: metav1.Now(),
			Message:            "All tasks ran successfully",
		})

		if err := r.Status().Update(context.Background(), pulp); err != nil && errors.IsConflict(err) {
			log.V(1).Info("Failed to update pulp status", "error", err)
			return ctrl.Result{Requeue: true}, nil
		}
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

	// we will only set .status.ingress_nginx:
	// - in the first execution (len==0)
	// - and if .spec.ingress_class_name is defined
	if len(pulp.Status.IngressClassName) == 0 && len(pulp.Spec.IngressClassName) > 0 {
		pulp.Status.IngressClassName = pulp.Spec.IngressClassName
		r.Status().Update(ctx, pulp)
	}

	return ctrl.Result{}, nil
}

// isDeploymentReady returns true if there is no unavailable Replicas for the deployment
func isDeploymentReady(deployment *appsv1.Deployment) bool {

	return deployment.Status.UnavailableReplicas <= 0
}

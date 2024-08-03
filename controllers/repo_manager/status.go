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
	"sync"
	"time"

	"github.com/go-logr/logr"
	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	"github.com/pulp/pulp-operator/controllers"
	"github.com/pulp/pulp-operator/controllers/settings"
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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// pulpResource contains the fields to update the .status.conditions from pulp instance
type pulpResource struct {
	Type          string
	Name          string
	ConditionType string
}

// pulpStatus will cheeck the READY state of the pods before considering the component status as ready
func (r *RepoManagerReconciler) pulpStatus(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp, log logr.Logger) *ctrl.Result {

	// update pulp.status.<fields>
	setStatusFields(ctx, pulp, *r)

	// update pulp.status.conditions[]
	if reconcile := r.setStatusConditions(ctx, pulp, log); reconcile != nil {
		return reconcile
	}

	return nil
}

// isDeploymentReady returns true if the deployment has the desired number of replicas in ready state
func isDeploymentReady(deployment *appsv1.Deployment) bool {
	return deployment.Status.ReadyReplicas == *deployment.Spec.Replicas && deployment.Status.UnavailableReplicas == 0
}

// setStatusConditions updates the pulp.Status.Conditions[] field
func (r *RepoManagerReconciler) setStatusConditions(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp, log logr.Logger) *reconcile.Result {
	pulpResources := []pulpResource{
		{
			Type:          string(settings.CONTENT),
			Name:          settings.CONTENT.DeploymentName(pulp.Name),
			ConditionType: cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Content-Ready",
		},
		{
			Type:          string(settings.API),
			Name:          settings.API.DeploymentName(pulp.Name),
			ConditionType: cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-API-Ready",
		},
		{
			Type:          string(settings.WORKER),
			Name:          settings.WORKER.DeploymentName(pulp.Name),
			ConditionType: cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Worker-Ready",
		},
		{
			Type:          string(settings.WEB),
			Name:          settings.WEB.DeploymentName(pulp.Name),
			ConditionType: cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Web-Ready",
		},
	}

	var wg sync.WaitGroup

	// each pulpcore status resource (content,worker,api) will be checked in a different go-routine to avoid
	// an issue with one of the status resource not getting updated until the previous one finishes
	for _, resource := range pulpResources {
		wg.Add(1)

		// if route or ingress we should do nothing
		if !r.needsIngressStatusUpdate(ctx, resource, pulp) {
			wg.Done()
			continue
		}

		go func(resource pulpResource) {
			defer wg.Done()
			deployment := &appsv1.Deployment{}
			typeCapitalized := cases.Title(language.English, cases.Compact).String(resource.Type)
			if err := r.Get(ctx, types.NamespacedName{Name: resource.Name, Namespace: pulp.Namespace}, deployment); err == nil {
				if !isDeploymentReady(deployment) {
					log.Info(resource.Name + " pods not ready yet ...")
					controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, resource.ConditionType, "Updating"+typeCapitalized+"Deployment", typeCapitalized+" deployment not ready yet")
				} else if v1.IsStatusConditionFalse(pulp.Status.Conditions, resource.ConditionType) {
					controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionTrue, resource.ConditionType, typeCapitalized+"TasksFinished", "All "+typeCapitalized+" tasks ran successfully")
					r.recorder.Event(pulp, corev1.EventTypeNormal, typeCapitalized+"Ready", "All "+typeCapitalized+" tasks ran successfully")
				}
			} else {
				log.Error(err, "Failed to get Pulp "+resource.Name+" Deployment")
			}
		}(resource)
	}
	wg.Wait()

	// requeue until all deployments get READY
	r.Get(ctx, types.NamespacedName{Name: pulp.Name, Namespace: pulp.Namespace}, pulp)
	for _, resource := range pulpResources {
		if v1.IsStatusConditionFalse(pulp.Status.Conditions, resource.ConditionType) {
			return &ctrl.Result{RequeueAfter: time.Second * 10}
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
			return &ctrl.Result{Requeue: true}
		}
		log.Info(pulp.Name + " finished execution ...")
	}
	return nil
}

// needsIngressStatusUpdate returns false when there is no need to deploy pulp-web, so we will not need to worry about updating .status field with it
func (r *RepoManagerReconciler) needsIngressStatusUpdate(ctx context.Context, resource pulpResource, pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool {
	if resource.Type == string(settings.WEB) {
		if isRoute(pulp) || r.isNginxIngress(pulp) {
			return false
		}
		if isIngress(pulp) {
			currentIngress := &netv1.Ingress{}
			r.Get(ctx, types.NamespacedName{Name: pulp.Name, Namespace: pulp.Namespace}, currentIngress)
			if currentIngress.Annotations["web"] == "false" {
				return false
			}
		}
	}
	return true
}

// setStatusFields updates all pulp.Status.<fields>
func setStatusFields(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp, r RepoManagerReconciler) {
	conditionFunctions := []struct {
		verifyFunc func(*repomanagerpulpprojectorgv1beta2.Pulp) bool
		fieldName  string
	}{
		{verifyFunc: deploymentTypeCondition(), fieldName: "DeploymentType"},
		{verifyFunc: objAzureSecretCondition(), fieldName: "ObjectStorageAzureSecret"},
		{verifyFunc: objS3SecretCondition(), fieldName: "ObjectStorageS3Secret"},
		{verifyFunc: dbFieldsEncrSecretCondition(), fieldName: "DBFieldsEncryptionSecret"},
		{verifyFunc: ingressTypeCondition(), fieldName: "IngressType"},
		{verifyFunc: containerTokenSecretCondition(), fieldName: "ContainerTokenSecret"},
		{verifyFunc: ingressClassNameCondition(), fieldName: "IngressClassName"},
		{verifyFunc: pulpSecretKeyCondition(), fieldName: "PulpSecretKey"},
	}

	for _, v := range conditionFunctions {
		setStatus(ctx, pulp, v.verifyFunc, r, v.fieldName)
	}

	/*
		corner cases where .status.<field> is not equals to .spec.<field>
	*/
	// update pulp image name status
	if controllers.ImageChanged(pulp) {
		pulp.Status.Image = pulp.Spec.Image + ":" + pulp.Spec.ImageVersion
		r.Status().Update(ctx, pulp)
	}

	// we will only set .status.external_cache_secret in the first execution (len==0)
	// and if .spec.external_cache_secret is defined
	checkCacheFunc := externalCacheSecretCondition()
	if checkCacheFunc(pulp) {
		pulp.Status.ExternalCacheSecret = pulp.Spec.Cache.ExternalCacheSecret
		r.Status().Update(ctx, pulp)
	}

	// update telemetry status
	checkTelemtryFunc := telemetryEnabledCondition()
	if checkTelemtryFunc(pulp) {
		pulp.Status.TelemetryEnabled = pulp.Spec.Telemetry.Enabled
		r.Status().Update(ctx, pulp)
	}

	// remove .status.allowed_content_checksums field in case it is not defined anymore
	checkContentChecksumsFunc := allowedContentChecksumsCondition()
	if checkContentChecksumsFunc(pulp) {
		pulp.Status.AllowedContentChecksums = ""
		r.Status().Update(ctx, pulp)
	}

}

// deploymentTypeCondition returns the function to verify if a new pulp.Status.DeploymentType should be set
func deploymentTypeCondition() func(*repomanagerpulpprojectorgv1beta2.Pulp) bool {
	return func(pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool { return len(pulp.Status.DeploymentType) == 0 }
}

// objAzureSecretCondition returns the function to verify if a new pulp.Status.ObjectStorageAzureSecret should be set
func objAzureSecretCondition() func(*repomanagerpulpprojectorgv1beta2.Pulp) bool {
	return func(pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool {
		return len(pulp.Status.ObjectStorageAzureSecret) == 0 && len(pulp.Spec.ObjectStorageAzureSecret) > 0
	}
}

// objS3SecretCondition returns the function to verify if a new pulp.Status.ObjectStorageS3Secret should be set
func objS3SecretCondition() func(*repomanagerpulpprojectorgv1beta2.Pulp) bool {
	return func(pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool {
		return len(pulp.Status.ObjectStorageS3Secret) == 0 && len(pulp.Spec.ObjectStorageS3Secret) > 0
	}
}

// dbFieldsEncrSecretCondition returns the function to verify if a new pulp.Status.DBFieldsEncryptionSecret should be set
func dbFieldsEncrSecretCondition() func(*repomanagerpulpprojectorgv1beta2.Pulp) bool {
	return func(pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool {
		return len(pulp.Status.DBFieldsEncryptionSecret) == 0 && len(pulp.Spec.DBFieldsEncryptionSecret) > 0
	}
}

// ingressTypeCondition returns the function to verify if a new pulp.Status.IngressType should be set
func ingressTypeCondition() func(*repomanagerpulpprojectorgv1beta2.Pulp) bool {
	return func(pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool { return len(pulp.Status.IngressType) == 0 }
}

// containerTokenSecretCondition returns the function to verify if a new pulp.Status.ContainerTokenSecret should be set
func containerTokenSecretCondition() func(*repomanagerpulpprojectorgv1beta2.Pulp) bool {
	return func(pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool {
		return len(pulp.Status.ContainerTokenSecret) == 0 && len(pulp.Spec.ContainerTokenSecret) > 0
	}
}

// adminPwdSecretCondition returns the function to verify if a new pulp.Status.AdminPasswordSecret should be set
func adminPwdSecretCondition() func(*repomanagerpulpprojectorgv1beta2.Pulp) bool {
	return func(pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool {
		return len(pulp.Status.AdminPasswordSecret) == 0 && len(pulp.Spec.AdminPasswordSecret) > 0
	}
}

// externalCacheSecretCondition returns the function to verify if a new pulp.Status.ExternalCacheSecret should be set
func externalCacheSecretCondition() func(*repomanagerpulpprojectorgv1beta2.Pulp) bool {
	return func(pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool {
		return len(pulp.Status.ExternalCacheSecret) == 0 && len(pulp.Spec.Cache.ExternalCacheSecret) > 0
	}
}

// ingressClassNameCondition returns the function to verify if a new pulp.Status.IngressClassName should be set
func ingressClassNameCondition() func(*repomanagerpulpprojectorgv1beta2.Pulp) bool {
	return func(pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool {
		return len(pulp.Status.IngressClassName) == 0 && len(pulp.Spec.IngressClassName) > 0
	}
}

// telemetryEnabledCondition returns the function to verify if a new pulp.Status.TelemetryEnabled should be set
func telemetryEnabledCondition() func(*repomanagerpulpprojectorgv1beta2.Pulp) bool {
	return func(pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool {
		return pulp.Status.TelemetryEnabled != pulp.Spec.Telemetry.Enabled
	}
}

// pulpSecretKeyCondition returns the function to verify if a new pulp.Status.PulpSecretKey should be set
func pulpSecretKeyCondition() func(*repomanagerpulpprojectorgv1beta2.Pulp) bool {
	return func(pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool {
		return len(pulp.Status.PulpSecretKey) == 0 && len(pulp.Spec.PulpSecretKey) > 0
	}
}

// allowedContentChecksumsCondition returns the function to verify if a new pulp.Status.AllowedContentChecksums should be set
func allowedContentChecksumsCondition() func(*repomanagerpulpprojectorgv1beta2.Pulp) bool {
	return func(pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool {
		return len(pulp.Spec.AllowedContentChecksums) == 0
	}
}

// setStatus updates an specific field from pulp.Status
func setStatus(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp, condition func(*repomanagerpulpprojectorgv1beta2.Pulp) bool, r RepoManagerReconciler, field string) {
	if condition(pulp) {
		currentSpec := reflect.ValueOf(pulp.Spec).FieldByName(field).String()
		reflect.ValueOf(&pulp.Status).Elem().FieldByName(field).SetString(currentSpec)
		r.Status().Update(ctx, pulp)
	}
}

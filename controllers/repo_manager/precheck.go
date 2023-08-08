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

	"github.com/go-logr/logr"
	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	"github.com/pulp/pulp-operator/controllers"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	v1 "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// prechecks verifies pulp cr fields inconsistencies
func prechecks(ctx context.Context, r *RepoManagerReconciler, pulp *repomanagerpulpprojectorgv1beta2.Pulp) (*ctrl.Result, error) {

	// initialize pulp .status.condition field
	if reconcile, err := initializeStatusCondition(ctx, r, pulp); err != nil {
		return &reconcile, err
	}

	// verify if pulp-web image version matches pulp-minimal image version
	if reconcile := checkImageVersion(r, pulp); reconcile != nil {
		return reconcile, nil
	}

	// verify if all expected ingress fields are defined
	if reconcile := checkIngressDefinition(r.RawLogger, pulp); reconcile != nil {
		return reconcile, nil
	}

	// verify and run ansible migration tasks
	if reconcile, err := ansibleMigration(ctx, r, pulp); err != nil {
		return &reconcile, err
	}

	// verify if multiple storage types were provided
	if reconcile := checkMultipleStorageType(r.RawLogger, pulp); reconcile != nil {
		return reconcile, nil
	}

	// verify if ingress_type==route in a non-ocp cluster
	if reconcile := checkRouteNotOCP(r.RawLogger, pulp); reconcile != nil {
		return reconcile, nil
	}

	// verify if a change in an immutable field has been tried
	if reconcile := checkImmutableFields(ctx, r, pulp); reconcile != nil {
		return reconcile, nil
	}

	// verify if all secrets defined in pulp cr are available
	if reconcile := checkSecretsAvailability(ctx, r, pulp); reconcile != nil {
		return reconcile, nil
	}

	return nil, nil
}

// initializeStatusCondition sets the .status.condition field with the initial value
func initializeStatusCondition(ctx context.Context, r *RepoManagerReconciler, pulp *repomanagerpulpprojectorgv1beta2.Pulp) (ctrl.Result, error) {
	log := r.RawLogger

	// "initialize" operator's .status.condition field
	if v1.FindStatusCondition(pulp.Status.Conditions, cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType)+"-Operator-Finished-Execution") == nil {
		log.V(1).Info("Creating operator's .status.conditions[] field ...")
		v1.SetStatusCondition(&pulp.Status.Conditions, metav1.Condition{
			Type:               cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Operator-Finished-Execution",
			Status:             metav1.ConditionFalse,
			Reason:             "OperatorRunning",
			LastTransitionTime: metav1.Now(),
			Message:            pulp.Name + " operator tasks running",
		})
		if err := r.Status().Update(ctx, pulp); err != nil {
			log.Error(err, "Failed to update operator's .status.conditions[] field!")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// checkImageVersion verifies if pulp-webcheckImageVersion image version matches pulp-minimal
func checkImageVersion(r *RepoManagerReconciler, pulp *repomanagerpulpprojectorgv1beta2.Pulp) *ctrl.Result {
	if r.needsPulpWeb(pulp) && pulp.Spec.ImageVersion != pulp.Spec.ImageWebVersion {
		if pulp.Spec.InhibitVersionConstraint {
			controllers.CustomZapLogger().Warn("image_version should be equal to image_web_version! Using different versions is not recommended and can make the application unreachable")
		} else {
			r.RawLogger.Error(nil, "image_version should be equal to image_web_version. Please, define image_version and image_web_version with the same value")
			return &ctrl.Result{}
		}
	}
	return nil
}

// checkIngressDefinition verifies if all ingress fields are defined when ingress_type==ingress
func checkIngressDefinition(log logr.Logger, pulp *repomanagerpulpprojectorgv1beta2.Pulp) *ctrl.Result {
	// in case of ingress_type == ingress.
	if isIngress(pulp) {

		// If ingress_type==ingress the operator should fail in case no ingress_class provided
		// To avoid errors with clusters configured without or with multiple default IngressClass we will ask users to pass an ingress_class
		if len(pulp.Spec.IngressClassName) == 0 {
			log.Error(nil, "ingress_type defined as ingress but no ingress_class_name provided. Please, define the ingress_class_name field (with the name of the IngressClass that the operator should use to deploy the new Ingress) to avoid unexpected errors with multiple controllers available")
			return &ctrl.Result{}
		}

		// the operator should also fail in case no ingress_host is provided
		// ingress_host is used to populate CONTENT_ORIGIN and ANSIBLE_API_HOSTNAME vars from settings.py
		// https://docs.pulpproject.org/pulpcore/configuration/settings.html#content-origin
		//   "A required string containing the protocol, fqdn, and port where the content app is reachable by users.
		//   This is used by pulpcore and various plugins when referring users to the content app."
		// pulp.Spec.Hostname is DEPRECATED! Temporarily adding it to keep compatibility with ansible version.
		if len(pulp.Spec.IngressHost) == 0 && len(pulp.Spec.Hostname) == 0 {
			log.Error(nil, "ingress_type defined as ingress but no ingress_host provided. Please, define the ingress_host field with the fqdn where "+pulp.Spec.DeploymentType+" should be accessed. This field is required to access API and also redirect "+pulp.Spec.DeploymentType+" CONTENT requests")
			return &ctrl.Result{}
		}
	}
	return nil
}

// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
func ansibleMigration(ctx context.Context, r *RepoManagerReconciler, pulp *repomanagerpulpprojectorgv1beta2.Pulp) (ctrl.Result, error) {
	if requeue, err := ansibleMigrationTasks(controllers.FunctionResources{Context: ctx, Client: r.Client, Pulp: pulp, Scheme: r.Scheme, Logger: r.RawLogger}); needsRequeue(err, requeue) {
		return ctrl.Result{Requeue: true}, err
	}
	return ctrl.Result{}, nil
}

// checkMultipleStorageType verifies if there is more than one storage type defined.
// Only a single type should be provided, if more the operator will not be able to
// determine which one should be used.
func checkMultipleStorageType(log logr.Logger, pulp *repomanagerpulpprojectorgv1beta2.Pulp) *ctrl.Result {
	for _, resource := range []string{controllers.PulpResource, controllers.CacheResource, controllers.DatabaseResource} {
		if foundMultiStorage, storageType := controllers.MultiStorageConfigured(pulp, resource); foundMultiStorage {
			log.Error(nil, "found more than one storage type \""+strings.Join(storageType, `", "`)+"\" for "+resource+". Please, choose only one storage type or do not define any to use emptyDir")
			return &ctrl.Result{}
		}
	}
	return nil
}

// checkRouteNotOCP verifies if this is an non-OCP cluster and "ingress_type: route".
func checkRouteNotOCP(log logr.Logger, pulp *repomanagerpulpprojectorgv1beta2.Pulp) *ctrl.Result {
	isOpenShift, _ := controllers.IsOpenShift()
	if !isOpenShift && isRoute(pulp) {
		log.Error(nil, "ingress_type is configured with route in a non-ocp environment. Please, choose another ingress_type (options: [ingress,nodeport]). Route resources are specific to OpenShift installations.")
		return &ctrl.Result{}
	}
	return nil
}

// checkImmutableFields verifies if an immutable field had changed
func checkImmutableFields(ctx context.Context, r *RepoManagerReconciler, pulp *repomanagerpulpprojectorgv1beta2.Pulp) *ctrl.Result {
	// Checking immutable fields update
	immutableFields := []immutableField{
		{FieldName: "DeploymentType", FieldPath: repomanagerpulpprojectorgv1beta2.PulpSpec{}},
		{FieldName: "ObjectStorageAzureSecret", FieldPath: repomanagerpulpprojectorgv1beta2.PulpSpec{}},
		{FieldName: "ObjectStorageS3Secret", FieldPath: repomanagerpulpprojectorgv1beta2.PulpSpec{}},
		{FieldName: "DBFieldsEncryptionSecret", FieldPath: repomanagerpulpprojectorgv1beta2.PulpSpec{}},
		{FieldName: "ContainerTokenSecret", FieldPath: repomanagerpulpprojectorgv1beta2.PulpSpec{}},
		{FieldName: "AdminPasswordSecret", FieldPath: repomanagerpulpprojectorgv1beta2.PulpSpec{}},
		{FieldName: "ExternalCacheSecret", FieldPath: repomanagerpulpprojectorgv1beta2.Cache{}},
	}
	for _, field := range immutableFields {
		// if tried to modify an immutable field we should trigger a reconcile loop
		if r.checkImmutableFields(ctx, pulp, field, r.RawLogger) {
			return &ctrl.Result{}
		}
	}
	return nil
}

// checkSecretsAvailability verifies if the secrets defined in Pulp CR are available.
// If an expected secret is not found, the operator will fail early and
// NOT trigger a reconciliation loop to avoid "spamming" error messages until
// the expected secret is found.
func checkSecretsAvailability(ctx context.Context, r *RepoManagerReconciler, pulp *repomanagerpulpprojectorgv1beta2.Pulp) *ctrl.Result {
	if err := checkSecretsAvailable(controllers.FunctionResources{Context: ctx, Client: r.Client, Pulp: pulp, Scheme: r.Scheme, Logger: r.RawLogger}); err != nil {
		r.RawLogger.Error(err, "Secret defined in Pulp CR not found!")
		return &ctrl.Result{}
	}
	return nil
}

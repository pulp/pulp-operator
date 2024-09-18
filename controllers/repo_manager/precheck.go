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
	"encoding/json"
	"strings"

	"github.com/go-logr/logr"
	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	"github.com/pulp/pulp-operator/controllers"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

	// verify inconsistency in file_storage_* definition
	if reconcile := checkFileStorage(ctx, r, pulp); reconcile != nil {
		return reconcile, nil
	}

	// verify inconsistency in allowed_content_checksums definition
	if reconcile := checkAllowedContentChecksums(ctx, r, pulp); reconcile != nil {
		return reconcile, nil
	}

	// verify if LDAP CA is provided in case settings.py expects it
	if reconcile := checkLDAPCA(ctx, r, pulp); reconcile != nil {
		return reconcile, nil
	}

	// verify the metadata signing definitions
	if reconcile := checkSigningScripts(ctx, r, pulp); reconcile != nil {
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
		if len(pulp.Spec.IngressHost) == 0 {
			log.Error(nil, "ingress_type defined as ingress but no ingress_host provided. Please, define the ingress_host field with the fqdn where "+pulp.Spec.DeploymentType+" should be accessed. This field is required to access API and also redirect "+pulp.Spec.DeploymentType+" CONTENT requests")
			return &ctrl.Result{}
		}
	}
	return nil
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

// checkFileStorage verifies if there is a file_storage definition but the storage_class is not provided
// the file_storage_* fields are used to provision the PVC using the provided file_storage_class
// if no file_storage_class is provided, the other fields will not be useful and can cause confusion
func checkFileStorage(ctx context.Context, r *RepoManagerReconciler, pulp *repomanagerpulpprojectorgv1beta2.Pulp) *ctrl.Result {
	if hasFileStorageDefinition(pulp) && len(pulp.Spec.FileStorageClass) == 0 {
		r.RawLogger.Error(nil, "No file_storage_class provided for the file_storage_{access_mode,size} definition(s)!")
		r.RawLogger.Error(nil, "Provide a file_storage_storage_class with the file_storage_{access_mode,size} fields to deploy Pulp with persistent data")
		r.RawLogger.Error(nil, "or remove all file_storage_* fields to deploy Pulp with emptyDir.")
		return &ctrl.Result{}
	}

	if len(pulp.Spec.FileStorageClass) > 0 && (len(pulp.Spec.FileStorageAccessMode) == 0 || len(pulp.Spec.FileStorageSize) == 0) {
		r.RawLogger.Error(nil, "file_storage_class provided but no file_storage_size and/or file_storage_access_mode defined!")
		r.RawLogger.Error(nil, "Provide a file_storage_size and file_storage_access_mode fields to deploy Pulp with persistent data")
		r.RawLogger.Error(nil, "or remove all file_storage_* fields to deploy Pulp with emptyDir.")
		return &ctrl.Result{}
	}
	return nil
}

// hasFileStorageDefinition returns true if any file_storage field is defined
func hasFileStorageDefinition(pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool {
	return len(pulp.Spec.FileStorageAccessMode) > 0 || len(pulp.Spec.FileStorageSize) > 0
}

// checkAllowedContentChecksums verifies the following conditions for allowed_content_checksums:
// * deprecated checksums algorithms
// * mandatory checksums present (for now, only sha256 is required)
// * checksums provided are valid
func checkAllowedContentChecksums(ctx context.Context, r *RepoManagerReconciler, pulp *repomanagerpulpprojectorgv1beta2.Pulp) *ctrl.Result {
	logger := controllers.CustomZapLogger()

	if len(pulp.Spec.AllowedContentChecksums) == 0 {
		return nil
	}

	for _, v := range pulp.Spec.AllowedContentChecksums {
		if ok := verifyChecksum(v, validContentChecksums); !ok {
			logger.Error("Checksum " + v + " is not valid!")
			return &ctrl.Result{}
		}

		if deprecated := verifyChecksum(v, deprecatedContentChecksum); deprecated {
			logger.Warn("Checksum " + v + " is deprecated by some Pulp plugins, it is not recommended using it in production.")
		}
	}

	if missing, ok := requiredContentChecksums(pulp.Spec.AllowedContentChecksums); !ok {
		missingJson, _ := json.Marshal(missing)
		logger.Error("Missing required checksum(s): " + string(missingJson))
		return &ctrl.Result{}
	}
	return nil
}

// checkLDAPCA verifies if there is a file provided in auth_ldap_ca_file (from pulp.Spec.LDAP.Config) field and if it does
// we need to ensure that .spec.LDAP.CA is provided
func checkLDAPCA(ctx context.Context, r *RepoManagerReconciler, pulp *repomanagerpulpprojectorgv1beta2.Pulp) *ctrl.Result {
	if len(pulp.Spec.LDAP.Config) == 0 {
		return nil
	}

	// retrieve the cert mountPoint from LDAP config Secret
	secretName := pulp.Spec.LDAP.Config
	secret := &corev1.Secret{}
	r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: pulp.Namespace}, secret)
	_, caDefined := secret.Data["auth_ldap_ca_file"]

	// if auth_ldap_ca is defined, but .spec.ldap.ca is not, abort because it
	// would fail to find the mount point and break the operator execution
	if !caDefined && len(pulp.Spec.LDAP.CA) > 0 {
		r.RawLogger.Error(nil, "auth_ldap_cafile is defined in "+pulp.Spec.LDAP.Config+" Secret, but no .spec.ldap.ca was found! Provide both values or none to avoid error in Pulp execution.")
		return &ctrl.Result{}
	}

	// if there is no CA definition we don't need more checks
	if !caDefined {
		return nil
	}

	// if there is a CA definition, we need to ensure that Pulp CR is defined
	// with the Secret to get it
	if len(pulp.Spec.LDAP.CA) == 0 {
		r.RawLogger.Error(nil, "The "+pulp.Spec.LDAP.Config+" Secret provided a configuration for the LDAP CA file (auth_ldap_ca_file field), but Pulp CR(.spec.LDAP.CA) does not have the Secret name to get it!")
		return &ctrl.Result{}
	}
	return nil
}

// checkSigningScripts verifies if signing_script and/or signing_secret is/are defined
func checkSigningScripts(ctx context.Context, r *RepoManagerReconciler, pulp *repomanagerpulpprojectorgv1beta2.Pulp) *ctrl.Result {
	if len(pulp.Spec.SigningScripts) > 0 && len(pulp.Spec.SigningSecret) == 0 {
		r.RawLogger.Error(nil, "spec.signing_scripts is defined but spec.signing_secret was not found! Provide both values or none to avoid error in Pulp execution.")
		return &ctrl.Result{}
	}
	if len(pulp.Spec.SigningScripts) == 0 && len(pulp.Spec.SigningSecret) > 0 {
		r.RawLogger.Error(nil, "spec.signing_secret is defined but spec.signing_scripts was not found! Provide both values or none to avoid error in Pulp execution.")
		return &ctrl.Result{}
	}

	return nil
}

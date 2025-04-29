package repo_manager

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	crypt_rand "crypto/rand"
	"crypto/x509"
	b64 "encoding/base64"
	"encoding/pem"
	"io"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	"github.com/pulp/pulp-operator/controllers"
	pulp_ocp "github.com/pulp/pulp-operator/controllers/ocp"
	"github.com/pulp/pulp-operator/controllers/settings"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Generate a random string with length pwdSize
func createPwd(pwdSize int) string {
	const chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	pwd := make([]byte, pwdSize)
	for i := range pwd {
		pwd[i] = chars[rand.Intn(len(chars))]
	}
	return string(pwd)
}

// Generate a random string to use as a django SECRET_KEY
func djangoKey() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*(-_=+)"
	pwd := make([]byte, 50)
	for i := range pwd {
		pwd[i] = chars[rand.Intn(len(chars))]
	}
	return string(pwd)
}

// sortKeys will return an ordered slice of strings defined with the keys from a.
// It is used to make sure that custom settings from pulp-server secret
// will be built in the same order and avoiding issues when verifying if its
// content is as expected. For example, this will avoid the controller having
// to check if
//
//	  pulp_settings:
//		    allowed_export_paths:
//		    - /tmp
//		    telemetry: falsew
//
// will converge into a secret like:
//
//	ALLOWED_IMPORT_PATHS = ["/tmp"]
//	TELEMETRY = "false"
//
// instead of (different order, which would fail the checkSecretModification)
//
//	TELEMETRY = "false"
//	ALLOWED_IMPORT_PATHS = ["/tmp"]
func sortKeys[V interface{} | []uint8](a map[string]V) []string {
	keys := make([]string, 0, len(a))
	for k := range a {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func genTokenAuthKey() (string, string) {
	newKey, _ := ecdsa.GenerateKey(elliptic.P256(), crypt_rand.Reader)
	pubKeyDER, _ := x509.MarshalPKIXPublicKey(&newKey.PublicKey)
	ecDER, _ := x509.MarshalECPrivateKey(newKey)

	privateKey := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: ecDER}))
	publicKey := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyDER}))

	return privateKey, publicKey
}

// checkSecretsAvailable verifies if the list of secrets that pulp-server secret can depend on
// are available.
func checkSecretsAvailable(funcResources controllers.FunctionResources) error {
	ctx := funcResources.Context
	pulp := funcResources.Pulp

	secrets := []string{"ObjectStorageAzureSecret", "ObjectStorageS3Secret", "SSOSecret"}
	for _, secretField := range secrets {
		structField := reflect.Indirect(reflect.ValueOf(pulp)).FieldByName("Spec").FieldByName(secretField)
		if structField.IsValid() && len(structField.Interface().(string)) != 0 {
			secret := &corev1.Secret{}
			if err := funcResources.Get(ctx, types.NamespacedName{Name: structField.Interface().(string), Namespace: pulp.Namespace}, secret); err != nil {
				return err
			}
		}
	}

	if len(pulp.Spec.Database.ExternalDBSecret) != 0 {
		secret := &corev1.Secret{}
		if err := funcResources.Get(ctx, types.NamespacedName{Name: pulp.Spec.Database.ExternalDBSecret, Namespace: pulp.Namespace}, secret); err != nil {
			return err
		}
	}

	if len(pulp.Spec.Cache.ExternalCacheSecret) != 0 {
		secret := &corev1.Secret{}
		if err := funcResources.Get(ctx, types.NamespacedName{Name: pulp.Spec.Cache.ExternalCacheSecret, Namespace: pulp.Namespace}, secret); err != nil {
			return err
		}
	}

	// LDAP Secrets
	if len(pulp.Spec.LDAP.Config) != 0 {
		secret := &corev1.Secret{}
		if err := funcResources.Get(ctx, types.NamespacedName{Name: pulp.Spec.LDAP.Config, Namespace: pulp.Namespace}, secret); err != nil {
			return err
		}
	}
	if len(pulp.Spec.LDAP.CA) != 0 {
		secret := &corev1.Secret{}
		if err := funcResources.Get(ctx, types.NamespacedName{Name: pulp.Spec.LDAP.CA, Namespace: pulp.Namespace}, secret); err != nil {
			return err
		}
	}

	return nil
}

// updateIngressType will check the current definition of ingress_type and will handle the different
// modification scenarios
func (r *RepoManagerReconciler) updateIngressType(ctx context.Context, pulp *pulpv1.Pulp) {

	// if pulp CR was defined with route and user modified it to anything else
	// delete all routes with operator's labels
	// remove route .status.conditions
	// update .status.ingress_type = nodeport
	if strings.ToLower(pulp.Status.IngressType) == "route" && !isRoute(pulp) {

		route := &routev1.Route{}
		routesLabelSelector := settings.CommonLabels(*pulp)
		listOpts := []client.DeleteAllOfOption{
			client.InNamespace(pulp.Namespace),
			client.MatchingLabels(routesLabelSelector),
		}
		r.DeleteAllOf(ctx, route, listOpts...)
		routeConditionType := "Pulp-Route-Ready"
		v1.RemoveStatusCondition(&pulp.Status.Conditions, routeConditionType)

		pulp.Status.IngressType = pulp.Spec.IngressType
		r.Status().Update(ctx, pulp)

		// nothing else to do (the controller will be responsible for setting up the other resources)
		return
	}

	// if pulp CR was defined with ingress and user modified it to anything else
	// delete all ingress with operator's labels
	// remove ingress .status.conditions
	if strings.ToLower(pulp.Status.IngressType) == "ingress" && strings.ToLower(pulp.Spec.IngressType) != "ingress" {

		ingress := &netv1.Ingress{}
		ingresssLabelSelector := settings.CommonLabels(*pulp)
		listOpts := []client.DeleteAllOfOption{
			client.InNamespace(pulp.Namespace),
			client.MatchingLabels(ingresssLabelSelector),
		}
		r.DeleteAllOf(ctx, ingress, listOpts...)
		ingressConditionType := "Pulp-Ingress-Ready"
		v1.RemoveStatusCondition(&pulp.Status.Conditions, ingressConditionType)

		pulp.Status.IngressType = pulp.Spec.IngressType

		if len(pulp.Status.IngressClassName) > 0 {
			pulp.Status.IngressClassName = ""
		}

		// remove pulp-web components
		controllers.RemovePulpWebResources(controllers.FunctionResources{Context: ctx, Client: r.Client, Pulp: pulp, Scheme: nil, Logger: logr.Logger{}})

		r.Status().Update(ctx, pulp)

		// nothing else to do (the controller will be responsible for setting up the other resources)
		return
	}

	// if pulp CR was defined with nodeport or loadbalancer and user modified it to anything else
	// delete all pulp-web resources
	// remove pulp-web .status.conditions
	// update .status.ingress_type
	// we will not remove configmap to avoid losing resources that are potentially unrecoverable
	if (strings.ToLower(pulp.Status.IngressType) == "nodeport" && strings.ToLower(pulp.Spec.IngressType) != "nodeport") ||
		(strings.ToLower(pulp.Status.IngressType) == "loadbalancer" && strings.ToLower(pulp.Spec.IngressType) != "loadbalancer") {

		// remove pulp-web components
		controllers.RemovePulpWebResources(controllers.FunctionResources{Context: ctx, Client: r.Client, Pulp: pulp, Scheme: nil, Logger: logr.Logger{}})

		pulp.Status.IngressType = pulp.Spec.IngressType
		r.Status().Update(ctx, pulp)

		// nothing else to do (the controller will be responsible for setting up the other resources)
		return
	}
}

// updateIngressClass will check the current definition of ingress_class_name and will handle the different
// modification scenarios
func (r *RepoManagerReconciler) updateIngressClass(ctx context.Context, pulp *pulpv1.Pulp) {

	// if the new one uses nginx controller
	if r.isNginxIngress(pulp) {

		// remove pulp-web components
		webDeployment := &appsv1.Deployment{}
		if err := r.Get(ctx, types.NamespacedName{Name: settings.WEB.DeploymentName(pulp.Name), Namespace: pulp.Namespace}, webDeployment); err == nil {
			r.Delete(ctx, webDeployment)
		}

		webSvc := &corev1.Service{}
		if err := r.Get(ctx, types.NamespacedName{Name: settings.PulpWebService(pulp.Name), Namespace: pulp.Namespace}, webSvc); err == nil {
			r.Delete(ctx, webSvc)
		}
		webConditionType := "Pulp-Web-Ready"
		v1.RemoveStatusCondition(&pulp.Status.Conditions, webConditionType)

		// or the new one does not use nginx controller anymore
	} else {
		// remove ingress resource
		ingress := &netv1.Ingress{}
		r.Get(ctx, types.NamespacedName{Name: pulp.Name, Namespace: pulp.Namespace}, ingress)
		r.Delete(ctx, ingress)

		ingressConditionType := "Pulp-Ingress-Ready"
		v1.RemoveStatusCondition(&pulp.Status.Conditions, ingressConditionType)
	}

	// handle OCP specific modifications on ingressclass change
	pulp_ocp.UpdateIngressClass(controllers.FunctionResources{Context: ctx, Client: r.Client, Pulp: pulp, Scheme: nil, Logger: logr.Logger{}})

	pulp.Status.IngressClassName = pulp.Spec.IngressClassName
	r.Status().Update(ctx, pulp)
}

// ResourceDefinition has the attributes of a Pulp Resource
type ResourceDefinition struct {
	// A Context carries a deadline, a cancellation signal, and other values across
	// API boundaries.
	context.Context
	// Type is used to define what Kubernetes resource should be provisioned
	Type interface{}
	// Name sets the resource name
	Name string
	// Alias is used in .status.conditions field
	Alias string
	// ConditionType is used to update .status.conditions with the current resource state
	ConditionType string
	// Pulp is the Schema for the pulps API
	*pulpv1.Pulp
}

// createPulpResource executes a set of instructions to provision Pulp resources
func (r *RepoManagerReconciler) createPulpResource(resource ResourceDefinition, createFunc func(controllers.FunctionResources) client.Object) (bool, error) {
	log := r.RawLogger
	var object client.Object
	objKind := ""

	// assert resource type
	switch resourceType := resource.Type.(type) {
	case *corev1.Secret:
		object = resourceType
		objKind = "Secret"
	case *appsv1.Deployment:
		object = resourceType
		objKind = "Deployment"
	case *corev1.Service:
		object = resourceType
		objKind = "Service"
	case *corev1.PersistentVolumeClaim:
		object = resourceType
		objKind = "PVC"
	case *corev1.ConfigMap:
		object = resourceType
		objKind = "ConfigMap"
	case *batchv1.CronJob:
		object = resourceType
		objKind = "CronJob"
	}

	// define the list of parameters to pass to "provisioner" function
	funcResources := controllers.FunctionResources{Context: resource.Context, Pulp: resource.Pulp, Logger: log, Scheme: r.Scheme, Client: r.Client}

	// set of instructions to create a resource (the following are almost the same for most of Pulp resources)
	// - we check if the resource exists
	// - if not we update Pulp CR status, add a log message and create the resource
	//   - if the resource provisioning fails we update the status, add an error message and return the error
	//   - if the resource is provisioned we create an event and return
	// - for any other error (besides resource not found) we add an error log and return error
	currentResource := object
	err := r.Get(resource.Context, types.NamespacedName{Name: resource.Name, Namespace: resource.Pulp.Namespace}, currentResource)
	if err != nil && k8s_errors.IsNotFound(err) {
		expectedResource := createFunc(funcResources)
		controllers.UpdateStatus(resource.Context, r.Client, resource.Pulp, metav1.ConditionFalse, resource.ConditionType, "Creating"+resource.Alias+objKind, "Creating "+resource.Name+" "+objKind)
		log.Info("Creating a new "+resource.Name+" "+objKind, "Namespace", resource.Pulp.Namespace, "Name", resource.Name)
		err = r.Create(resource.Context, expectedResource)

		if err != nil {
			log.Error(err, "Failed to create new "+resource.Name+" "+objKind)
			controllers.UpdateStatus(resource.Context, r.Client, resource.Pulp, metav1.ConditionFalse, resource.ConditionType, "ErrorCreating"+resource.Alias+objKind, "Failed to create "+resource.Name+" "+objKind+": "+err.Error())
			r.recorder.Event(resource.Pulp, corev1.EventTypeWarning, "Failed", "Failed to create a new "+resource.Name+" "+objKind)
			return false, err
		}
		r.recorder.Event(resource.Pulp, corev1.EventTypeNormal, "Created", resource.Name+" "+objKind+" created")
		return true, nil
	} else if err != nil {
		log.Error(err, "Failed to get "+resource.Name+" "+objKind)
		return false, err
	}

	return false, nil
}

// createFernetKey creates a random key that will be used in "database_fields.symmetric.key"
func createFernetKey() string {
	key := [32]byte{}
	io.ReadFull(crypt_rand.Reader, key[:])
	return b64.StdEncoding.EncodeToString(key[:])
}

// needsRequeue will return true if the controller should trigger a new reconcile loop
func needsRequeue(err error, pulpController ctrl.Result) bool {
	return err != nil || !reflect.DeepEqual(pulpController, ctrl.Result{})
}

// needsPulpWeb will return true if ingress_type is not route and the ingress_type provided does not
// support nginx controller, which is a scenario where pulp-web should be deployed
func (r *RepoManagerReconciler) needsPulpWeb(pulp *pulpv1.Pulp) bool {
	return !isRoute(pulp) && !controllers.IsNginxIngressSupported(pulp)
}

// isNginxIngress will check if ingress_type is defined as "ingress"
func isIngress(pulp *pulpv1.Pulp) bool {
	return strings.ToLower(pulp.Spec.IngressType) == "ingress"
}

// isRoute will check if ingress_type is defined as "route"
func isRoute(pulp *pulpv1.Pulp) bool {
	return strings.ToLower(pulp.Spec.IngressType) == "route"
}

// isNginxIngress returns true if pulp is defined with ingress_type==ingress and the controller of the ingresclass provided is a nginx
func (r *RepoManagerReconciler) isNginxIngress(pulp *pulpv1.Pulp) bool {
	return isIngress(pulp) && controllers.IsNginxIngressSupported(pulp)
}

// getRootURL handles user facing URLs
func getRootURL(pulp pulpv1.Pulp) string {
	scheme := "https"
	if isIngress(&pulp) {
		if pulp.Spec.IngressTLSSecret == "" {
			scheme = "http"
		}
		hostname := pulp.Spec.IngressHost
		return scheme + "://" + hostname
	}
	if isRoute(&pulp) {
		return "https://" + pulp_ocp.GetRouteHost(&pulp)
	}

	return "http://" + settings.PulpWebService(pulp.Name) + "." + pulp.Namespace + ".svc.cluster.local:24880"
}

// ignoreUpdateCRStatusPredicate filters update events on pulpbackup CR status
func ignoreCronjobStatus() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObject := e.ObjectOld.(*batchv1.CronJob)
			newObject := e.ObjectNew.(*batchv1.CronJob)

			// if old cronjob.status != new cronjob.status return false, which will instruct
			// the controller to do nothing on this update event
			return reflect.DeepEqual(oldObject.Status, newObject.Status)
		},
	}
}

// findPulpDependentObjects will search for Pulp objects based on Secret/ConfigMap names defined in Pulp CR.
// It is used to "link" these Secrets/ConfigMaps (not "owned" by Pulp operator) with Pulp object
func (r *RepoManagerReconciler) findPulpDependentObjects(ctx context.Context, obj client.Object) []reconcile.Request {

	associatedPulp := pulpv1.PulpList{}
	opts := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("objects", obj.GetName()),
		Namespace:     obj.GetNamespace(),
	}
	if err := r.List(ctx, &associatedPulp, opts); err != nil {
		return []reconcile.Request{}
	}
	if len(associatedPulp.Items) > 0 {
		return []reconcile.Request{{
			NamespacedName: types.NamespacedName{
				Name:      associatedPulp.Items[0].GetName(),
				Namespace: associatedPulp.Items[0].GetNamespace(),
			},
		}}
	}

	return []reconcile.Request{}
}

// restartPulpCorePods will redeploy all pulpcore (API,content,worker) pods.
func (r *RepoManagerReconciler) restartPulpCorePods(ctx context.Context, pulp *pulpv1.Pulp) {
	r.RawLogger.Info("Reprovisioning pulpcore pods to get the new settings ...")
	pulp.Status.LastDeploymentUpdate = time.Now().Format(time.RFC3339)
	r.Status().Update(ctx, pulp)
}

// runMigration deploys a k8s Job to run django migrations in case of pulpcore image change
func (r *RepoManagerReconciler) runMigration(ctx context.Context, pulp *pulpv1.Pulp) {
	if !r.needsMigration(ctx, pulp) {
		return
	}
	r.migrationJob(ctx, pulp)
}

// needsMigration verifies if the pulpcore image has changed and no migration
// has been done yet.
func (r *RepoManagerReconciler) needsMigration(ctx context.Context, pulp *pulpv1.Pulp) bool {

	// if migrations are disabled in spec, we should not run it even if
	// the image has changed
	if pulp.Spec.DisableMigrations {
		return false
	}

	// run a migration if the storage type changed
	if controllers.StorageTypeChanged(pulp) {
		// we need to update the status now to avoid a new reconciliation
		// recreating a new job
		pulp.Status.StorageType = controllers.GetStorageType(*pulp)[0]
		r.Status().Update(ctx, pulp)
		return true
	}

	return controllers.ImageChanged(pulp) && !r.migrationDone(ctx, pulp)
}

// migrationDone checks if there is a migration Job with the expected image
func (r *RepoManagerReconciler) migrationDone(ctx context.Context, pulp *pulpv1.Pulp) bool {
	jobList := &batchv1.JobList{}
	labels := jobLabels(*pulp)
	listOpts := []client.ListOption{
		client.InNamespace(pulp.Namespace),
		client.MatchingLabels(labels),
	}

	r.List(ctx, jobList, listOpts...)
	return hasActiveJob(*jobList, pulp)
}

// jobImageEqualsCurrent verifies if the image used in migration job is the same
// as the one used in pulpcore-{api,content,worker} pods
func jobImageEqualsCurrent(job batchv1.Job, pulp *pulpv1.Pulp) bool {
	return job.Spec.Template.Spec.Containers[0].Image == pulp.Spec.Image+":"+pulp.Spec.ImageVersion
}

// hasActiveJob will iterate over the JobList looking for any Job with the current
// pulpcore image (meaning that a migration for the current version has already
// been triggered and there is no need to create a new job).
func hasActiveJob(jobList batchv1.JobList, pulp *pulpv1.Pulp) bool {
	for _, job := range jobList.Items {
		if jobImageEqualsCurrent(job, pulp) {
			return true
		}
	}
	return false
}

// validContentChecksums returns a map of the checksums algorithms supported and
// required by Pulp (only sha256 is required in the current Pulp version).
func validContentChecksums() map[string]bool {
	return map[string]bool{
		"md5":    false,
		"sha1":   false,
		"sha224": false,
		"sha256": true,
		"sha384": false,
		"sha512": false,
	}
}

// requiredContentChecksums verifies if all the required checksums are in the
// checksums list provided and, if not, returns the list of missing checksums
func requiredContentChecksums(cs []string) ([]string, bool) {
	missing := []string{}
	currentChecksums := map[string]bool{}
	for _, v := range cs {
		currentChecksums[v] = true
	}

	for checksum, required := range validContentChecksums() {
		if !required {
			continue
		}
		if _, ok := currentChecksums[checksum]; !ok {
			missing = append(missing, checksum)
		}
	}

	if len(missing) > 0 {
		return missing, false
	}
	return []string{}, true
}

// deprecatedContentChecksum returns a map of the checksums algorithms that are
// supported by Pulp but deprecated by some Pulp plugin.
func deprecatedContentChecksum() map[string]bool {
	return map[string]bool{
		"md5":  true,
		"sha1": true,
	}
}

// verifyChecksum verifies if the provided checksum algorithm is supported and/or
// deprecated (depending on the validateFunction provided).
func verifyChecksum(checksum string, validateFunction func() map[string]bool) bool {
	_, isValid := validateFunction()[checksum]
	return isValid
}

// runSigningScriptJob deploys a k8s Job to store the metadata signing scripts
func (r *RepoManagerReconciler) runSigningScriptJob(ctx context.Context, pulp *pulpv1.Pulp) {
	if len(pulp.Spec.SigningScripts) == 0 {
		return
	}

	if !r.secretModified(ctx, pulp.Spec.SigningScripts, pulp.Namespace) {
		return
	}
	r.signingScriptJob(ctx, pulp)
}

// secretModified verifies if the secret has been modified based on the hash stored
func (r *RepoManagerReconciler) secretModified(ctx context.Context, secretName, namespace string) bool {
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, secret); err != nil {
		r.RawLogger.Error(err, "Failed to find "+secretName+" Secret!")
	}

	calculatedHash := controllers.CalculateHash(secret.Data)
	currentHash := controllers.GetCurrentHash(secret)
	return currentHash != calculatedHash
}

// runSigningSecretTasks restart pulpcore pods if the signing secret has been modified
func (r *RepoManagerReconciler) runSigningSecretTasks(ctx context.Context, pulp *pulpv1.Pulp) *ctrl.Result {
	secretName := pulp.Spec.SigningSecret
	if len(secretName) == 0 {
		return nil
	}

	if !r.secretModified(ctx, secretName, pulp.Namespace) {
		return nil
	}

	r.restartPulpCorePods(ctx, pulp)
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: pulp.Namespace}, secret); err != nil {
		r.RawLogger.Error(err, "Failed to find "+secretName+" Secret!")
	}

	r.RawLogger.V(1).Info("Updating " + secretName + " hash label ...")
	calculatedHash := controllers.CalculateHash(secret.Data)
	controllers.SetHashLabel(calculatedHash, secret)
	if err := r.Update(ctx, secret); err != nil {
		r.RawLogger.Error(err, "Failed to update "+secretName+" Secret label!")
	}

	return &ctrl.Result{Requeue: true}
}

// TODO: the ipv6 incompatibility should be handled by oci-image.
// Remove this function after updating the image.
// postgresConnectionConfigMap creates a ConfigMap with a script to verify the
// conectivity with Postgres
func postgresConnectionConfigMap(resources controllers.FunctionResources) client.Object {
	pulp := resources.Pulp
	checkPostgres := map[string]string{
		"wait_on_postgres.py": `#!/usr/bin/env python3
import os
import socket
import sys
import time
if __name__ == "__main__":
    postgres_is_alive = False
    s4 = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    tries = 0
    print("Waiting on postgresql to start...")
    while not postgres_is_alive and tries < 100:
        tries += 1
        pg_port = 5432
        try:
            env_port = os.environ.get("POSTGRES_SERVICE_PORT", "5432")
            pg_port = int(env_port)
        except ValueError:
            pass
        try:
            print("Checking postgres host %s" % os.environ["POSTGRES_SERVICE_HOST"])
            print("Checking postgres port %s" % os.environ["POSTGRES_SERVICE_PORT"])
            s4.connect((os.environ["POSTGRES_SERVICE_HOST"], pg_port))
        except socket.error:
            time.sleep(3)
        else:
            postgres_is_alive = True

    if postgres_is_alive:
        print("Postgres started!")
        sys.exit(0)
    else:
        print("Unable to reach postgres on port %s" % os.environ["POSTGRES_SERVICE_PORT"])
        sys.exit(1)`,
	}
	labels := settings.CommonLabels(*pulp)
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      settings.PulpWorkerProbe(pulp.Name),
			Namespace: pulp.Namespace,
			Labels:    labels,
		},
		Data: checkPostgres,
	}
}

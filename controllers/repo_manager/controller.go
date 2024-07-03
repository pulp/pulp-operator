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
	"strings"

	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	"github.com/pulp/pulp-operator/controllers"
	pulp_ocp "github.com/pulp/pulp-operator/controllers/ocp"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	policy "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// RepoManagerReconciler reconciles a Pulp object
type RepoManagerReconciler struct {
	client.Client
	RawLogger  logr.Logger
	RESTClient rest.Interface
	RESTConfig *rest.Config
	Scheme     *runtime.Scheme
	recorder   record.EventRecorder
}

//+kubebuilder:rbac:groups=repo-manager.pulpproject.org,namespace=pulp-operator-system,resources=pulps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=repo-manager.pulpproject.org,namespace=pulp-operator-system,resources=pulps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=repo-manager.pulpproject.org,namespace=pulp-operator-system,resources=pulps/finalizers,verbs=update
//+kubebuilder:rbac:groups=networking.k8s.io,namespace=pulp-operator-system,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=route.openshift.io,namespace=pulp-operator-system,resources=routes;routes/custom-host,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,namespace=pulp-operator-system,resources=roles;rolebindings,verbs=create;update;patch;delete;watch;get;list
//+kubebuilder:rbac:groups=core,namespace=pulp-operator-system,resources=pods;pods/log;serviceaccounts;configmaps;secrets;services;persistentvolumeclaims,verbs=create;update;patch;delete;watch;get;list
//+kubebuilder:rbac:groups=core,namespace=pulp-operator-system,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=apps,namespace=pulp-operator-system,resources=deployments;statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=policy,namespace=pulp-operator-system,resources=poddisruptionbudgets,verbs=get;list;create;delete;patch;update;watch
//+kubebuilder:rbac:groups=batch,namespace=pulp-operator-system,resources=cronjobs;jobs,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *RepoManagerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.RawLogger

	pulp := &repomanagerpulpprojectorgv1beta2.Pulp{}
	err := r.Get(ctx, req.NamespacedName, pulp)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Pulp resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Pulp")
		return ctrl.Result{}, err
	}

	// if Unmanaged the operator should do nothing
	// this is useful in situations where we don't want the operator to do reconciliation
	// for example, during a troubleshooting or for testing
	if pulp.Spec.Unmanaged {
		return ctrl.Result{}, nil
	}

	// create RH pull secret and CA configmap (if needed)
	if reconcile, err := ocpTasks(ctx, pulp, *r); err != nil || reconcile != nil {
		return *reconcile, err
	}

	// run multiple validations before deploying pulp resources
	if reconcile, err := prechecks(ctx, r, pulp); err != nil || reconcile != nil {
		return *reconcile, err
	}

	// Create ServiceAccount
	if reconcile, err := r.CreateServiceAccount(ctx, pulp); needsRequeue(err, reconcile) {
		return reconcile, err
	}

	if reconcile, err := databaseTasks(ctx, pulp, *r); err != nil || reconcile != nil {
		return *reconcile, err
	}

	if reconcile, err := cacheTasks(ctx, pulp, *r); err != nil || reconcile != nil {
		return *reconcile, err
	}

	if reconcile, err := pulpCoreTasks(ctx, pulp, *r); err != nil || reconcile != nil {
		return *reconcile, err
	}

	log.V(1).Info("Running status tasks")
	if reconcile := r.pulpStatus(ctx, pulp, log); reconcile != nil {
		return *reconcile, nil
	}

	// If we get into here it means that there is no reconciliation
	// nor controller tasks pending
	log.Info("Operator tasks synced")
	return ctrl.Result{}, nil
}

func ocpTasks(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp, r RepoManagerReconciler) (*ctrl.Result, error) {
	if isOpenShift, _ := controllers.IsOpenShift(); !isOpenShift {
		return nil, nil
	}

	if err := createRHPullSecret(ctx, pulp, r); err != nil {
		return &ctrl.Result{}, err
	}

	if reconcile, err := injectCertificates(ctx, pulp, r); err != nil || reconcile != nil {
		return reconcile, err
	}

	return nil, nil
}

func createRHPullSecret(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp, r RepoManagerReconciler) error {
	r.RawLogger.V(1).Info("Running on OpenShift cluster")
	if err := pulp_ocp.CreateRHOperatorPullSecret(r.Client, ctx, *pulp); err != nil {
		return err
	}

	return nil
}

func injectCertificates(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp, r RepoManagerReconciler) (*ctrl.Result, error) {
	if !pulp.Spec.TrustedCa {
		return nil, nil
	}

	pulpController, err := pulp_ocp.CreateEmptyConfigMap(r.Client, r.Scheme, ctx, pulp, r.RawLogger)
	if needsRequeue(err, pulpController) {
		return &pulpController, err
	}

	return nil, nil
}

func databaseTasks(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp, r RepoManagerReconciler) (*ctrl.Result, error) {
	log := r.RawLogger

	// Do not provision postgres resources if using external DB
	if len(pulp.Spec.Database.ExternalDBSecret) != 0 {
		return nil, nil
	}

	log.V(1).Info("Running database tasks")
	pulpController, err := r.databaseController(ctx, pulp, log)
	if needsRequeue(err, pulpController) {
		return &pulpController, err
	}
	return nil, nil
}

func pulpCoreTasks(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp, r RepoManagerReconciler) (*ctrl.Result, error) {
	log := r.RawLogger

	log.V(1).Info("Running file storage tasks ...")
	if pulpController, err := r.pulpFileStorage(ctx, pulp); pulpController != nil || err != nil {
		return pulpController, err
	}

	log.V(1).Info("Running secrets tasks ...")
	if pulpController, err := r.createSecrets(ctx, pulp); pulpController != nil || err != nil {
		return pulpController, err
	}

	log.V(1).Info("Running API tasks")
	if pulpController, err := r.pulpApiController(ctx, pulp, log); needsRequeue(err, pulpController) {
		return &pulpController, err
	}

	// create the job to run django migrations
	r.runMigration(ctx, pulp)

	// create the job to store the metadata signing scripts
	r.runSigningScriptJob(ctx, pulp)
	if pulpController := r.runSigningSecretTasks(ctx, pulp); pulpController != nil {
		return pulpController, nil
	}

	log.V(1).Info("Running content tasks")
	if pulpController, err := r.pulpContentController(ctx, pulp, log); needsRequeue(err, pulpController) {
		return &pulpController, err
	}

	log.V(1).Info("Running worker tasks")
	if pulpController, err := r.pulpWorkerController(ctx, pulp, log); needsRequeue(err, pulpController) {
		return &pulpController, err
	}

	// create the job to reset pulp admin password in case admin_password_secret has changed
	r.updateAdminPasswordJob(ctx, pulp)

	// create the job to update the allowed_content_checksums
	r.updateContentChecksumsJob(ctx, pulp)

	// if this is the first reconciliation loop (.status.ingress_type == "") OR
	// if there is no update in ingressType field
	if len(pulp.Status.IngressType) == 0 || pulp.Status.IngressType == pulp.Spec.IngressType {
		if isRoute(pulp) {
			log.V(1).Info("Running route tasks")
			pulpController, err := pulp_ocp.PulpRouteController(controllers.FunctionResources{Context: ctx, Client: r.Client, Pulp: pulp, Scheme: r.Scheme, Logger: log}, r.RESTClient, r.RESTConfig)
			if needsRequeue(err, pulpController) {
				return &pulpController, err
			}
		} else if isIngress(pulp) {
			log.V(1).Info("Running ingress tasks")
			pulpController, err := r.pulpIngressController(ctx, pulp, log)
			if needsRequeue(err, pulpController) {
				return &pulpController, err
			}
		} else {
			log.V(1).Info("Running web tasks")
			pulpController, err := r.pulpWebController(ctx, pulp, log)
			if needsRequeue(err, pulpController) {
				return &pulpController, err
			}
		}
	} else {
		r.updateIngressType(ctx, pulp)
		return &ctrl.Result{Requeue: true}, nil
	}

	// if this is the first reconciliation loop (.status.ingress_class_name == "") OR
	// if there is no update in ingressType field
	if !strings.EqualFold(pulp.Status.IngressClassName, pulp.Spec.IngressClassName) {
		r.updateIngressClass(ctx, pulp)
		return &ctrl.Result{Requeue: true}, nil
	}

	log.V(1).Info("Running PDB tasks")
	if pulpController, err := r.pdbController(ctx, pulp, log); needsRequeue(err, pulpController) {
		return &pulpController, err
	}

	if pulpController, err := r.galaxy(controllers.FunctionResources{Context: ctx, Client: r.Client, Pulp: pulp, Scheme: r.Scheme, Logger: log}); needsRequeue(err, pulpController) {
		return &pulpController, err
	}

	// remove telemetry resources in case it is not enabled anymore
	if pulp.Status.TelemetryEnabled && !pulp.Spec.Telemetry.Enabled {
		controllers.RemoveTelemetryResources(controllers.FunctionResources{Context: ctx, Client: r.Client, Pulp: pulp, Scheme: r.Scheme, Logger: log})
	}

	return nil, nil
}

func cacheTasks(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp, r RepoManagerReconciler) (*ctrl.Result, error) {
	log := r.RawLogger

	// Provision redis resources only if
	// - no external cache cluster provided
	// - cache is enabled
	if len(pulp.Spec.Cache.ExternalCacheSecret) == 0 && pulp.Spec.Cache.Enabled {
		log.V(1).Info("Running cache tasks")
		pulpController, err := r.pulpCacheController(ctx, pulp, log)
		if needsRequeue(err, pulpController) {
			return &pulpController, err
		}

	}

	// remove redis resources if cache is not enabled anymore
	if managedCacheDisabled(pulp) {
		pulpController, err := r.deprovisionCache(ctx, pulp, log)
		if needsRequeue(err, pulpController) {
			return &pulpController, err
		}
	}

	return nil, nil
}

// indexerFunc knows how to take an object and turn it into a series of non-namespaced keys
func indexerFunc(obj client.Object) []string {
	pulp := obj.(*repomanagerpulpprojectorgv1beta2.Pulp)
	var keys []string

	secrets := []string{"ObjectStorageAzureSecret", "ObjectStorageS3Secret", "SSOSecret", "AdminPasswordSecret", "PulpSecretKey", "SigningScripts", "SigningSecret"}
	for _, secretField := range secrets {
		structField := reflect.Indirect(reflect.ValueOf(pulp)).FieldByName("Spec").FieldByName(secretField).String()
		if structField != "" {
			keys = append(keys, structField)
		}
	}
	if pulp.Spec.Database.ExternalDBSecret != "" {
		keys = append(keys, pulp.Spec.Database.ExternalDBSecret)
	}
	if pulp.Spec.Cache.ExternalCacheSecret != "" {
		keys = append(keys, pulp.Spec.Cache.ExternalCacheSecret)
	}
	if pulp.Spec.LDAP.Config != "" {
		keys = append(keys, pulp.Spec.LDAP.Config)
	}
	if pulp.Spec.LDAP.CA != "" {
		keys = append(keys, pulp.Spec.LDAP.CA)
	}
	if customSettings := pulp.Spec.CustomPulpSettings; customSettings != "" {
		keys = append(keys, customSettings)
	}

	return keys
}

// SetupWithManager sets up the controller with the Manager.
func (r *RepoManagerReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// creates a new eventRecorder to be able to interact with events
	r.recorder = mgr.GetEventRecorderFor("Pulp")

	// adds an index to `object_storage_azure_secret` allowing to lookup `Pulp` by a referenced `Azure Object Storage Secret` name
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &repomanagerpulpprojectorgv1beta2.Pulp{}, "objects", indexerFunc); err != nil {
		return err
	}

	controller := ctrl.NewControllerManagedBy(mgr).
		For(&repomanagerpulpprojectorgv1beta2.Pulp{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&policy.PodDisruptionBudget{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&batchv1.CronJob{}, builder.WithPredicates(ignoreCronjobStatus())).
		Owns(&netv1.Ingress{}).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.findPulpDependentObjects),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(
			&corev1.ConfigMap{},
			handler.EnqueueRequestsFromMapFunc(r.findPulpDependentObjects),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		)

	if isOpenShift, _ := controllers.IsOpenShift(); isOpenShift {
		return controller.
			Owns(&routev1.Route{}).
			Complete(r)
	}

	return controller.Complete(r)
}

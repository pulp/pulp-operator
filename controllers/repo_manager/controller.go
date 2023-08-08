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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
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

	isOpenShift, _ := controllers.IsOpenShift()
	if isOpenShift {
		log.V(1).Info("Running on OpenShift cluster")
		if err := pulp_ocp.CreateRHOperatorPullSecret(r.Client, ctx, req.NamespacedName.Namespace); err != nil {
			return ctrl.Result{}, err
		}
	}

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

	// run multiple validations before deploying pulp resources
	if reconcile, err := prechecks(ctx, r, pulp); err != nil || reconcile != nil {
		return *reconcile, err
	}

	var pulpController reconcile.Result

	// Create an empty ConfigMap in which CNO will inject custom CAs
	if isOpenShift && pulp.Spec.TrustedCa {
		pulpController, err = pulp_ocp.CreateEmptyConfigMap(r.Client, r.Scheme, ctx, pulp, log)
		if needsRequeue(err, pulpController) {
			return pulpController, err
		}
	}

	// Create ServiceAccount
	pulpController, err = r.CreateServiceAccount(ctx, pulp)
	if needsRequeue(err, pulpController) {
		return pulpController, err
	}

	// Do not provision postgres resources if using external DB
	if len(pulp.Spec.Database.ExternalDBSecret) == 0 {
		log.V(1).Info("Running database tasks")
		pulpController, err = r.databaseController(ctx, pulp, log)
		if needsRequeue(err, pulpController) {
			return pulpController, err
		}
	}

	// Provision redis resources only if
	// - no external cache cluster provided
	// - cache is enabled
	if len(pulp.Spec.Cache.ExternalCacheSecret) == 0 && pulp.Spec.Cache.Enabled {
		log.V(1).Info("Running cache tasks")
		pulpController, err = r.pulpCacheController(ctx, pulp, log)
		if needsRequeue(err, pulpController) {
			return pulpController, err
		}

		// remove redis resources if cache is not enabled
	} else {
		pulpController, err = r.deprovisionCache(ctx, pulp, log)
		if needsRequeue(err, pulpController) {
			return pulpController, err
		}
	}

	log.V(1).Info("Running API tasks")
	pulpController, err = r.pulpApiController(ctx, pulp, log)
	if needsRequeue(err, pulpController) {
		return pulpController, err
	}

	// create the job to run django migrations
	r.runMigration(ctx, pulp)

	log.V(1).Info("Running content tasks")
	pulpController, err = r.pulpContentController(ctx, pulp, log)
	if needsRequeue(err, pulpController) {
		return pulpController, err
	}
	log.V(1).Info("Running worker tasks")
	pulpController, err = r.pulpWorkerController(ctx, pulp, log)
	if needsRequeue(err, pulpController) {
		return pulpController, err
	}

	// create the job to reset pulp admin password in case admin_password_secret has changed
	r.updateAdminPasswordJob(ctx, pulp)

	// if this is the first reconciliation loop (.status.ingress_type == "") OR
	// if there is no update in ingressType field
	if len(pulp.Status.IngressType) == 0 || pulp.Status.IngressType == pulp.Spec.IngressType {
		if isRoute(pulp) {
			log.V(1).Info("Running route tasks")
			pulpController, err = pulp_ocp.PulpRouteController(controllers.FunctionResources{Context: ctx, Client: r.Client, Pulp: pulp, Scheme: r.Scheme, Logger: log}, r.RESTClient, r.RESTConfig)
			if needsRequeue(err, pulpController) {
				return pulpController, err
			}
		} else if isIngress(pulp) {
			log.V(1).Info("Running ingress tasks")
			pulpController, err = r.pulpIngressController(ctx, pulp, log)
			if needsRequeue(err, pulpController) {
				return pulpController, err
			}
		} else {
			log.V(1).Info("Running web tasks")
			pulpController, err = r.pulpWebController(ctx, pulp, log)
			if needsRequeue(err, pulpController) {
				return pulpController, err
			}
		}
	} else {
		r.updateIngressType(ctx, pulp)
		return ctrl.Result{Requeue: true}, nil
	}

	// if this is the first reconciliation loop (.status.ingress_class_name == "") OR
	// if there is no update in ingressType field
	if !strings.EqualFold(pulp.Status.IngressClassName, pulp.Spec.IngressClassName) {
		r.updateIngressClass(ctx, pulp)
		return ctrl.Result{Requeue: true}, nil
	}

	log.V(1).Info("Running PDB tasks")
	pulpController, err = r.pdbController(ctx, pulp, log)
	if needsRequeue(err, pulpController) {
		return pulpController, err
	}

	if pulpController, err := r.galaxy(controllers.FunctionResources{Context: ctx, Client: r.Client, Pulp: pulp, Scheme: r.Scheme, Logger: log}); needsRequeue(err, pulpController) {
		return pulpController, err
	}

	// remove telemetry resources in case it is not enabled anymore
	if pulp.Status.TelemetryEnabled && !pulp.Spec.Telemetry.Enabled {
		controllers.RemoveTelemetryResources(controllers.FunctionResources{Context: ctx, Client: r.Client, Pulp: pulp, Scheme: r.Scheme, Logger: log})
	}

	log.V(1).Info("Running status tasks")
	pulpController, err = r.pulpStatus(ctx, pulp, log)
	if needsRequeue(err, pulpController) {
		return pulpController, err
	}

	// If we get into here it means that there is no reconciliation
	// nor controller tasks pending
	log.Info("Operator tasks synced")
	return ctrl.Result{}, nil
}

// indexerFunc knows how to take an object and turn it into a series of non-namespaced keys
func indexerFunc(obj client.Object) []string {
	pulp := obj.(*repomanagerpulpprojectorgv1beta2.Pulp)
	var keys []string

	secrets := []string{"ObjectStorageAzureSecret", "ObjectStorageS3Secret", "SSOSecret", "AdminPasswordSecret"}
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
	return keys
}

// SetupWithManager sets up the controller with the Manager.
func (r *RepoManagerReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// creates a new eventRecorder to be able to interact with events
	r.recorder = mgr.GetEventRecorderFor("Pulp")

	// adds an index to `object_storage_azure_secret` allowing to lookup `Pulp` by a referenced `Azure Object Storage Secret` name
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &repomanagerpulpprojectorgv1beta2.Pulp{}, "secrets", indexerFunc); err != nil {
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
			&source.Kind{Type: &corev1.Secret{}},
			handler.EnqueueRequestsFromMapFunc(r.findPulpDependentSecrets),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		)

	if isOpenShift, _ := controllers.IsOpenShift(); isOpenShift {
		return controller.
			Owns(&routev1.Route{}).
			Complete(r)
	}

	return controller.Complete(r)
}

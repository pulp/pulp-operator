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
	"fmt"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	policy "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	repomanagerv1alpha1 "github.com/pulp/pulp-operator/api/v1alpha1"
	"github.com/pulp/pulp-operator/controllers"
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
//+kubebuilder:rbac:groups=apps;networking.k8s.io,namespace=pulp-operator-system,resources=deployments;statefulsets;ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.openshift.io,resources=ingresses,verbs=get;list;watch
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingressclasses,verbs=get;list;watch
//+kubebuilder:rbac:groups=route.openshift.io,namespace=pulp-operator-system,resources=routes;routes/custom-host,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,namespace=pulp-operator-system,resources=pods;pods/log,verbs=get;list;
//+kubebuilder:rbac:groups=core;rbac.authorization.k8s.io,namespace=pulp-operator-system,resources=roles;rolebindings;serviceaccounts,verbs=create;update;patch;delete;watch;get;list;
//+kubebuilder:rbac:groups=core,namespace=pulp-operator-system,resources=configmaps;secrets;services;persistentvolumeclaims,verbs=create;update;patch;delete;watch;get;list;
//+kubebuilder:rbac:groups="",namespace=pulp-operator-system,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=policy,namespace=pulp-operator-system,resources=poddisruptionbudgets,verbs=get;list;create;delete;patch;update;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *RepoManagerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.RawLogger

	IsOpenShift, _ := controllers.IsOpenShift()
	if IsOpenShift {
		log.V(1).Info("Running on OpenShift cluster")
	}

	// Get redhat-operators-pull-secret
	defaultSecret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: "redhat-operators-pull-secret", Namespace: req.NamespacedName.Namespace}, defaultSecret)

	// Create the secret in case it is not found
	if err != nil && errors.IsNotFound(err) {
		defaultSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "redhat-operators-pull-secret",
				Namespace: req.NamespacedName.Namespace,
			},
			StringData: map[string]string{
				"operator": "pulp",
			},
		}
		r.Create(ctx, defaultSecret)
	} else if err != nil {
		log.Error(err, "Failed to get redhat-operators-pull-secret")
		return ctrl.Result{}, err
	}

	pulp := &repomanagerv1alpha1.Pulp{}
	err = r.Get(ctx, req.NamespacedName, pulp)

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

	needsPulpWeb := strings.ToLower(pulp.Spec.IngressType) != "route" && !controllers.IsNginxIngressSupported(r, pulp.Spec.IngressClassName)
	if needsPulpWeb && pulp.Spec.ImageVersion != pulp.Spec.ImageWebVersion {
		err := fmt.Errorf("image version and image web version should be equal ")
		log.Error(err, "ImageVersion should be equal to ImageWebVersion")
		return ctrl.Result{}, err
	}

	// If ingress_type==ingress the operator should fail in case no ingress_class provided
	// To avoid errors with clusters configured without or with multiple default IngressClass we will ask users to pass an ingress_class
	// in case of ingress_type == ingress.
	if isIngress(pulp) {
		if len(pulp.Spec.IngressClassName) == 0 {
			log.Error(nil, "ingress_type defined as ingress but no ingress_class_name provided.")
			return ctrl.Result{}, fmt.Errorf("please, define the ingress_class_name field (with the name of the IngressClass that the operator should use to deploy the new Ingress) to avoid unexpected errors with multiple controllers available")
		}
	}

	// Checking if there is more than one storage type defined.
	// Only a single type should be provided, if more the operator will not be able to
	// determine which one should be used.
	for _, resource := range []string{controllers.PulpResource, controllers.CacheResource, controllers.DatabaseResource} {
		if foundMultiStorage, storageType := controllers.MultiStorageConfigured(pulp, resource); foundMultiStorage {
			err := fmt.Errorf("found more than one storage type (%s) for %s", storageType, resource)
			log.Error(err, "Please choose only one storage type or do not define any to use emptyDir")
			return ctrl.Result{}, err
		}
	}

	// Check if this is an OCP cluster and "ingress_type: route".
	if !IsOpenShift && strings.ToLower(pulp.Spec.IngressType) == "route" {
		err := fmt.Errorf("ingress_type is configured with route in a non-ocp environment")
		log.Error(err, "Please choose another ingress_type (options: [ingress,nodeport]). Route resources are specific to OpenShift installations.")
		return ctrl.Result{}, err
	}

	// Checking immutable fields update
	immutableFields := []immutableField{
		{FieldName: "DeploymentType", FieldPath: repomanagerv1alpha1.PulpSpec{}},
		{FieldName: "ObjectStorageAzureSecret", FieldPath: repomanagerv1alpha1.PulpSpec{}},
		{FieldName: "ObjectStorageS3Secret", FieldPath: repomanagerv1alpha1.PulpSpec{}},
		{FieldName: "DBFieldsEncryptionSecret", FieldPath: repomanagerv1alpha1.PulpSpec{}},
		{FieldName: "ContainerTokenSecret", FieldPath: repomanagerv1alpha1.PulpSpec{}},
		{FieldName: "AdminPasswordSecret", FieldPath: repomanagerv1alpha1.PulpSpec{}},
		{FieldName: "ExternalCacheSecret", FieldPath: repomanagerv1alpha1.Cache{}},
	}
	for _, field := range immutableFields {
		// if tried to modify an immutable field we should trigger a reconcile loop
		if r.checkImmutableFields(ctx, pulp, field, log) {
			return ctrl.Result{}, nil
		}
	}

	var pulpController reconcile.Result

	// Create an empty ConfigMap in which CNO will inject custom CAs
	if IsOpenShift && pulp.Spec.TrustedCa {
		pulpController, err = r.createEmptyConfigMap(ctx, pulp, log)
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

	// if this is the first reconciliation loop (.status.ingress_type == "") OR
	// if there is no update in ingressType field
	if len(pulp.Status.IngressType) == 0 || pulp.Status.IngressType == pulp.Spec.IngressType {
		if strings.ToLower(pulp.Spec.IngressType) == "route" {
			log.V(1).Info("Running route tasks")
			pulpController, err = r.pulpRouteController(ctx, pulp, log)
			if needsRequeue(err, pulpController) {
				return pulpController, err
			}
		} else if isIngress(pulp) {
			log.V(1).Info("Running ingress tasks")
			pulpController, err = r.pulpIngressController(ctx, pulp, log)
			if needsRequeue(err, pulpController) {
				return pulpController, err
			}
		}
		if needsPulpWeb {
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

// SetupWithManager sets up the controller with the Manager.
func (r *RepoManagerReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// creates a new eventRecorder to be able to interact with events
	r.recorder = mgr.GetEventRecorderFor("Pulp")

	controller := ctrl.NewControllerManagedBy(mgr).
		For(&repomanagerv1alpha1.Pulp{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&policy.PodDisruptionBudget{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&netv1.Ingress{})

	if IsOpenShift, _ := controllers.IsOpenShift(); IsOpenShift {
		return controller.
			Owns(&routev1.Route{}).
			Complete(r)
	}

	return controller.Complete(r)
}

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

package pulp

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	repomanagerv1alpha1 "github.com/pulp/pulp-operator/api/v1alpha1"
	"github.com/pulp/pulp-operator/controllers"
)

// PulpReconciler reconciles a Pulp object
type PulpReconciler struct {
	client.Client
	RESTClient rest.Interface
	RESTConfig *rest.Config
	Scheme     *runtime.Scheme
	recorder   record.EventRecorder
}

//+kubebuilder:rbac:groups=repo-manager.pulpproject.org,resources=pulps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=repo-manager.pulpproject.org,resources=pulps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=repo-manager.pulpproject.org,resources=pulps/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps;networking.k8s.io,resources=deployments;statefulsets;ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.openshift.io,resources=ingresses,verbs=get;list;watch
//+kubebuilder:rbac:groups=route.openshift.io,resources=routes;routes/custom-host,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;
//+kubebuilder:rbac:groups=core,resources=configmaps;secrets;services;persistentvolumeclaims,verbs=create;update;patch;delete;watch;get;list;
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *PulpReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)
	IsOpenShift, _ := controllers.IsOpenShift()
	if IsOpenShift {
		log.Info("Running on OpenShift cluster")
	}

	pulp := &repomanagerv1alpha1.Pulp{}
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

	if len(pulp.Spec.ObjectStorageAzureSecret) > 0 && len(pulp.Spec.ObjectStorageS3Secret) > 0 {
		err := fmt.Errorf("only one object storage is allowed")
		log.Error(err, "Please choose between Azure and S3")
		return ctrl.Result{}, err
	}

	var pulpController reconcile.Result

	// Do not provision postgres resources if using external DB
	if reflect.DeepEqual(pulp.Spec.Database.ExternalDB, repomanagerv1alpha1.ExternalDB{}) {
		log.Info("Running database tasks")
		pulpController, err = r.databaseController(ctx, pulp, log)
		if err != nil {
			return pulpController, err
		} else if pulpController.Requeue {
			return pulpController, nil
		} else if pulpController.RequeueAfter > 0 {
			return pulpController, nil
		}
	}

	log.Info("Running cache tasks")
	pulpController, err = r.pulpCacheController(ctx, pulp, log)
	if err != nil {
		return pulpController, err
	} else if pulpController.Requeue {
		return pulpController, nil
	} else if pulpController.RequeueAfter > 0 {
		return pulpController, nil
	}

	log.Info("Running API tasks")
	pulpController, err = r.pulpApiController(ctx, pulp, log)
	if err != nil {
		return pulpController, err
	} else if pulpController.Requeue {
		return pulpController, nil
	} else if pulpController.RequeueAfter > 0 {
		return pulpController, nil
	}

	log.Info("Running content tasks")
	pulpController, err = r.pulpContentController(ctx, pulp, log)
	if err != nil {
		return pulpController, err
	} else if pulpController.Requeue {
		return pulpController, nil
	} else if pulpController.RequeueAfter > 0 {
		return pulpController, nil
	}

	log.Info("Running worker tasks")
	pulpController, err = r.pulpWorkerController(ctx, pulp, log)
	if err != nil {
		return pulpController, err
	} else if pulpController.Requeue {
		return pulpController, nil
	} else if pulpController.RequeueAfter > 0 {
		return pulpController, nil
	}

	if strings.ToLower(pulp.Spec.IngressType) == "route" {
		log.Info("Running route tasks")
		pulpController, err = r.pulpRouteController(ctx, pulp, log)
		if err != nil {
			return pulpController, err
		} else if pulpController.Requeue {
			return pulpController, nil
		} else if pulpController.RequeueAfter > 0 {
			return pulpController, nil
		}
	} else {
		log.Info("Running web tasks")
		pulpController, err = r.pulpWebController(ctx, pulp, log)
		if err != nil {
			return pulpController, err
		} else if pulpController.Requeue {
			return pulpController, nil
		} else if pulpController.RequeueAfter > 0 {
			return pulpController, nil
		}
	}

	log.Info("Running status tasks")
	pulpController, err = r.pulpStatus(ctx, pulp, log)
	if err != nil {
		return pulpController, err
	} else if pulpController.Requeue {
		return pulpController, nil
	}

	return pulpController, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PulpReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// creates a new eventRecorder to be able to interact with events
	r.recorder = mgr.GetEventRecorderFor("Pulp")

	return ctrl.NewControllerManagedBy(mgr).
		For(&repomanagerv1alpha1.Pulp{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

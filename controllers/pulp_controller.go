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

package controllers

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
)

// PulpReconciler reconciles a Pulp object
type PulpReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=repo-manager.pulpproject.org,resources=pulps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=repo-manager.pulpproject.org,resources=pulps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=repo-manager.pulpproject.org,resources=pulps/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments;statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;
//+kubebuilder:rbac:groups=core,resources=configmaps;secrets;services;persistentvolumeclaims,verbs=create;update;patch;delete;watch;get;list;

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Pulp object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *PulpReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

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

	pulpController, err := r.databaseController(ctx, pulp, log)
	if err != nil {
		return pulpController, err
	} else if pulpController.Requeue {
		return pulpController, nil
	} else if pulpController.RequeueAfter > 0 {
		return pulpController, nil
	}

	pulpController, err = r.pulpCacheController(ctx, pulp, log)
	if err != nil {
		return pulpController, err
	} else if pulpController.Requeue {
		return pulpController, nil
	} else if pulpController.RequeueAfter > 0 {
		return pulpController, nil
	}

	pulpController, err = r.pulpApiController(ctx, pulp, log)
	if err != nil {
		return pulpController, err
	} else if pulpController.Requeue {
		return pulpController, nil
	} else if pulpController.RequeueAfter > 0 {
		return pulpController, nil
	}

	pulpController, err = r.pulpContentController(ctx, pulp, log)
	if err != nil {
		return pulpController, err
	} else if pulpController.Requeue {
		return pulpController, nil
	} else if pulpController.RequeueAfter > 0 {
		return pulpController, nil
	}

	pulpController, err = r.pulpWorkerController(ctx, pulp, log)
	if err != nil {
		return pulpController, err
	} else if pulpController.Requeue {
		return pulpController, nil
	} else if pulpController.RequeueAfter > 0 {
		return pulpController, nil
	}

	pulpController, err = r.pulpWebController(ctx, pulp, log)
	if err != nil {
		return pulpController, err
	} else if pulpController.Requeue {
		return pulpController, nil
	} else if pulpController.RequeueAfter > 0 {
		return pulpController, nil
	}

	return pulpController, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PulpReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&repomanagerv1alpha1.Pulp{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

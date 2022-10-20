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
	"time"

	"github.com/go-logr/logr"
	repomanagerv1alpha1 "github.com/pulp/pulp-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8s_error "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	policy "k8s.io/api/policy/v1"
)

// pdbController creates and reconciles {api,content,worker,web} pdbs
func (r *RepoManagerReconciler) pdbController(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, log logr.Logger) (ctrl.Result, error) {

	pdbList := map[string]*policy.PodDisruptionBudgetSpec{
		"api":       pulp.Spec.Api.PDB,
		"content":   pulp.Spec.Content.PDB,
		"worker":    pulp.Spec.Worker.PDB,
		"webserver": pulp.Spec.Web.PDB,
	}

	for component, pdb := range pdbList {

		pdbFound := &policy.PodDisruptionBudget{}
		err := r.Get(ctx, types.NamespacedName{Name: component + "-pdb", Namespace: pulp.Namespace}, pdbFound)

		// check if PDB is defined
		// we need to check if pdb != nil (no .Spec.<component>.PDB field defined)
		// we also need to check if .Spec.<component>.PDB field is defined but with no content. For example:
		// api:
		//    pdb: {}
		if pdb != nil && !reflect.DeepEqual(pdb, &policy.PodDisruptionBudgetSpec{}) {

			// add label selector to PDBSpec
			// even though it is possible to pass a selector through PodDisruptionBudgetSpec we will overwrite
			// any config passed through pulp CR with the following
			pdb.Selector = &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/component": component,
				},
			}
			expectedPDB := &policy.PodDisruptionBudget{
				ObjectMeta: metav1.ObjectMeta{
					Name:      component + "-pdb",
					Namespace: pulp.Namespace,
				},
				Spec: *pdb,
			}
			ctrl.SetControllerReference(pulp, expectedPDB, r.Scheme)

			// Create PDB if not found
			if err != nil && k8s_error.IsNotFound(err) {
				log.Info("Creating a new " + component + " PDB ...")
				err = r.Create(ctx, expectedPDB)
				if err != nil {
					log.Error(err, "Failed to create new "+component+" PDB")
					return ctrl.Result{}, err
				}
				// PDB created successfully - return and requeue
				return ctrl.Result{Requeue: true}, nil
			} else if err != nil {
				log.Error(err, "Failed to get "+component+" PDB")
				return ctrl.Result{}, err
			}

			// Reconcile PDB
			if !equality.Semantic.DeepDerivative(expectedPDB.Spec, pdbFound.Spec) {
				log.Info("The " + component + "PDB has been modified! Reconciling ...")

				// I'm not sure why the error:
				// "metadata.resourceVersion: Invalid value: 0x0: must be specified for an update"
				// is happening when trying to update expectedPDB (this is not happening with the other resources)
				// this will set the pdb resourceversion with the current version
				expectedPDB.SetResourceVersion(pdbFound.GetResourceVersion())
				err = r.Update(ctx, expectedPDB)
				if err != nil {
					log.Error(err, "Error trying to update the "+component+" PDB object ... ")
					return ctrl.Result{}, err
				}
				return ctrl.Result{Requeue: true, RequeueAfter: time.Second}, nil
			}

			// and finally we need to check if pdb == nil || pdb == {} to remove any PDB resource
			// previously created but removed from Pulp CR
		} else {
			// if PDB is not found it means that it has been removed already, so nothing to do
			if err != nil && k8s_error.IsNotFound(err) {
				continue
			} else if err != nil {
				log.Error(err, "Failed to get "+component+" PDB")
				return ctrl.Result{}, err
			}

			r.Delete(ctx, pdbFound)
		}
	}

	return ctrl.Result{}, nil
}

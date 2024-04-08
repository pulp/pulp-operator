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

	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	"github.com/pulp/pulp-operator/controllers"
	"github.com/pulp/pulp-operator/controllers/settings"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// pulpFileStorage will provision a PVC when spec.file_storage_storage_class is defined
func (r *RepoManagerReconciler) pulpFileStorage(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp) (*ctrl.Result, error) {
	if !storageClassProvided(pulp) {
		return nil, nil
	}

	conditionType := cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-API-Ready"
	if requeue, err := r.createPulpResource(ResourceDefinition{ctx, &corev1.PersistentVolumeClaim{}, settings.DefaultPulpFileStorage(pulp.Name), "FileStorage", conditionType, pulp}, fileStoragePVC); err != nil {
		return &ctrl.Result{}, err
	} else if requeue {
		return &ctrl.Result{Requeue: true}, nil
	}

	return nil, nil
}

// fileStoragePVC returns a PVC object
func fileStoragePVC(resources controllers.FunctionResources) client.Object {

	pulp := resources.Pulp
	labels := settings.CommonLabels(*pulp)
	labels["app.kubernetes.io/component"] = "storage"
	// Define the new PVC
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      settings.DefaultPulpFileStorage(pulp.Name),
			Namespace: pulp.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): resource.MustParse(pulp.Spec.FileStorageSize),
				},
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.PersistentVolumeAccessMode(pulp.Spec.FileStorageAccessMode),
			},
			StorageClassName: &pulp.Spec.FileStorageClass,
		},
	}

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(pulp, pvc, resources.Scheme)
	return pvc
}

// storageClassProvided returns true if a StorageClass is provided in Pulp CR
func storageClassProvided(pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool {
	_, storageType := controllers.MultiStorageConfigured(pulp, "Pulp")
	return storageType[0] == controllers.SCNameType
}

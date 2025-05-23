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
	"regexp"

	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	"github.com/pulp/pulp-operator/controllers"
	"github.com/pulp/pulp-operator/controllers/settings"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *RepoManagerReconciler) CreateServiceAccount(ctx context.Context, pulp *pulpv1.Pulp) (ctrl.Result, error) {
	log := r.RawLogger
	conditionType := getApiConditionType()

	serviceAccountName := settings.PulpServiceAccount(pulp.Name)
	sa := &corev1.ServiceAccount{}
	err := r.Get(ctx, types.NamespacedName{Name: serviceAccountName, Namespace: pulp.Namespace}, sa)
	expectedSA := r.pulpSA(pulp)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating "+serviceAccountName+" ServiceAccount", "Namespace", expectedSA.Namespace, "Name", serviceAccountName)
		controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, conditionType, "CreatingSA", "Creating "+serviceAccountName+" SA resource")
		err = r.Create(ctx, expectedSA)
		if err != nil {
			log.Error(err, "Failed to create "+serviceAccountName+" ServiceAccount", "Namespace", expectedSA.Namespace, "Name", serviceAccountName)
			controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, conditionType, "ErrorCreatingSA", "Failed to create "+serviceAccountName+" SA: "+err.Error())
			r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to create "+serviceAccountName+" SA")
			return ctrl.Result{}, err
		}
		// SA created successfully - return and requeue
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Created", serviceAccountName+" SA created")
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get "+serviceAccountName+" SA")
		return ctrl.Result{}, err
	}

	// add the internalRegistrySecret to the list of imagePullSecrets
	internalRegistrySecret := r.getInternalRegistrySecret(ctx, serviceAccountName, pulp.Namespace)
	if internalRegistrySecret != "" {
		expectedSA.ImagePullSecrets = append([]corev1.LocalObjectReference{{Name: internalRegistrySecret}}, expectedSA.ImagePullSecrets...)
	}

	// Check and reconcile pulp-sa
	// Temporarily disabling to prevent an infinite reconciliation loop issue in OCP 4.16.
	/* 	if saModified(sa, expectedSA) {
		log.Info("The " + serviceAccountName + " SA has been modified! Reconciling ...")
		err = r.Update(ctx, expectedSA)
		if err != nil {
			log.Error(err, "Error trying to update "+serviceAccountName+" SA!")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	} */

	return r.CreateRole(ctx, pulp)
}

func (r *RepoManagerReconciler) CreateRole(ctx context.Context, pulp *pulpv1.Pulp) (ctrl.Result, error) {
	log := r.RawLogger
	conditionType := getApiConditionType()
	role := &rbacv1.Role{}
	err := r.Get(ctx, types.NamespacedName{Name: pulp.Name, Namespace: pulp.Namespace}, role)
	expectedRole := r.pulpRole(pulp)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Pulp Role", "Namespace", expectedRole.Namespace, "Name", expectedRole.Name)
		controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, conditionType, "CreatingRole", "Creating "+pulp.Name+" Role resource")
		err = r.Create(ctx, expectedRole)
		if err != nil {
			log.Error(err, "Failed to create new Pulp ServiceAccount", "Namespace", expectedRole.Namespace, "Name", expectedRole.Name)
			controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, conditionType, "ErrorCreatingRole", "Failed to create "+pulp.Name+" Role: "+err.Error())
			r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to create new Pulp Role")
			return ctrl.Result{}, err
		}
		// Role created successfully - return and requeue
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Created", "Pulp Role created")
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Pulp Role")
		return ctrl.Result{}, err
	}
	return r.CreateRoleBinding(ctx, pulp)
}

func (r *RepoManagerReconciler) CreateRoleBinding(ctx context.Context, pulp *pulpv1.Pulp) (ctrl.Result, error) {
	log := r.RawLogger
	conditionType := getApiConditionType()
	rolebinding := &rbacv1.RoleBinding{}
	err := r.Get(ctx, types.NamespacedName{Name: pulp.Name, Namespace: pulp.Namespace}, rolebinding)
	expectedRoleBinding := r.pulpRoleBinding(pulp)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Pulp RoleBinding", "Namespace", expectedRoleBinding.Namespace, "Name", expectedRoleBinding.Name)
		controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, conditionType, "CreatingRoleBinding", "Creating "+pulp.Name+" RoleBinding resource")
		err = r.Create(ctx, expectedRoleBinding)
		if err != nil {
			log.Error(err, "Failed to create new Pulp ServiceAccount", "Namespace", expectedRoleBinding.Namespace, "Name", expectedRoleBinding.Name)
			controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, conditionType, "ErrorCreatingRoleBinding", "Failed to create "+pulp.Name+" RoleBinding: "+err.Error())
			r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to create new Pulp RoleBinding")
			return ctrl.Result{}, err
		}
		// RoleBinding created successfully - return and requeue
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Created", "Pulp RoleBinding created")
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Pulp RoleBinding")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *RepoManagerReconciler) pulpSA(m *pulpv1.Pulp) *corev1.ServiceAccount {
	var imagePullSecrets []corev1.LocalObjectReference

	for _, pullSecret := range m.Spec.ImagePullSecrets {
		imagePullSecrets = append(imagePullSecrets, corev1.LocalObjectReference{Name: pullSecret})
	}

	annotations := m.Spec.SAAnnotations
	labels := m.Spec.SALabels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["app.kubernetes.io/name"] = m.Name + "-sa"
	labels["app.kubernetes.io/part-of"] = "pulp"

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        settings.PulpServiceAccount(m.Name),
			Namespace:   m.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		ImagePullSecrets: imagePullSecrets,
	}

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(m, sa, r.Scheme)
	return sa
}

// getInternalRegistrySecret gets the imagePullSecret for the internal registry that is created
// and added to the SA in OCP environments based on pattern:
//
//	<operator_instance_name>-dockercfg-<hash>
func (r *RepoManagerReconciler) getInternalRegistrySecret(ctx context.Context, saName, saNamespace string) string {
	sa := &corev1.ServiceAccount{}
	r.Get(ctx, types.NamespacedName{Name: saName, Namespace: saNamespace}, sa)
	for _, imagePullSecret := range sa.ImagePullSecrets {
		if match, _ := regexp.MatchString(saName+"-dockercfg-([a-z0-9]){5}", imagePullSecret.Name); match {
			return imagePullSecret.Name
		}
	}

	return ""
}

func (r *RepoManagerReconciler) pulpRole(m *pulpv1.Pulp) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":    m.Name + "-role",
				"app.kubernetes.io/part-of": "pulp",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods/log"},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "create", "delete"},
			},
		},
	}
}

func (r *RepoManagerReconciler) pulpRoleBinding(m *pulpv1.Pulp) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":    m.Name + "-rolebinding",
				"app.kubernetes.io/part-of": "pulp",
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: settings.PulpServiceAccount(m.Name),
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     m.Name,
		},
	}
}

// getConditionType returns a string with the .status.conditions.type from API resource
func getApiConditionType() string {
	return "Pulp-API-Ready"
}

// saModified returns true if some specific fields from a SA differs from the expected
/* func saModified(currentSA, expectedSA *corev1.ServiceAccount) bool {
	return !reflect.DeepEqual(currentSA.ImagePullSecrets, expectedSA.ImagePullSecrets) ||
		!reflect.DeepEqual(currentSA.ObjectMeta.Annotations, expectedSA.ObjectMeta.Annotations) ||
		!reflect.DeepEqual(currentSA.ObjectMeta.Labels, expectedSA.ObjectMeta.Labels)
}
*/

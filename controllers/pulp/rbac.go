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

	repomanagerv1alpha1 "github.com/pulp/pulp-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *PulpReconciler) CreateServiceAccount(ctx context.Context, pulp *repomanagerv1alpha1.Pulp) (ctrl.Result, error) {
	log := r.RawLogger
	sa := &corev1.ServiceAccount{}
	err := r.Get(ctx, types.NamespacedName{Name: pulp.Name, Namespace: pulp.Namespace}, sa)
	expectedSA := r.pulpSA(pulp)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new "+pulp.Spec.DeploymentType+" ServiceAccount", "Namespace", expectedSA.Namespace, "Name", expectedSA.Name)
		r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-API-Ready", "CreatingSA", "Creating "+pulp.Name+" SA resource")
		err = r.Create(ctx, expectedSA)
		if err != nil {
			log.Error(err, "Failed to create new "+pulp.Spec.DeploymentType+" ServiceAccount", "Namespace", expectedSA.Namespace, "Name", expectedSA.Name)
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-API-Ready", "ErrorCreatingSA", "Failed to create "+pulp.Name+" SA: "+err.Error())
			r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to create new "+pulp.Spec.DeploymentType+" SA")
			return ctrl.Result{}, err
		}
		// SA created successfully - return and requeue
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Created", pulp.Spec.DeploymentType+" SA created")
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get "+pulp.Spec.DeploymentType+" SA")
		return ctrl.Result{}, err
	}

	return r.CreateRole(ctx, pulp)
}

func (r *PulpReconciler) CreateRole(ctx context.Context, pulp *repomanagerv1alpha1.Pulp) (ctrl.Result, error) {
	log := r.RawLogger
	role := &rbacv1.Role{}
	err := r.Get(ctx, types.NamespacedName{Name: pulp.Name, Namespace: pulp.Namespace}, role)
	expectedRole := r.pulpRole(pulp)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new "+pulp.Spec.DeploymentType+" Role", "Namespace", expectedRole.Namespace, "Name", expectedRole.Name)
		r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-API-Ready", "CreatingRole", "Creating "+pulp.Name+" Role resource")
		err = r.Create(ctx, expectedRole)
		if err != nil {
			log.Error(err, "Failed to create new "+pulp.Spec.DeploymentType+" ServiceAccount", "Namespace", expectedRole.Namespace, "Name", expectedRole.Name)
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-API-Ready", "ErrorCreatingRole", "Failed to create "+pulp.Name+" Role: "+err.Error())
			r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to create new "+pulp.Spec.DeploymentType+" Role")
			return ctrl.Result{}, err
		}
		// Role created successfully - return and requeue
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Created", pulp.Spec.DeploymentType+" Role created")
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get "+pulp.Spec.DeploymentType+" Role")
		return ctrl.Result{}, err
	}
	return r.CreateRoleBinding(ctx, pulp)
}

func (r *PulpReconciler) CreateRoleBinding(ctx context.Context, pulp *repomanagerv1alpha1.Pulp) (ctrl.Result, error) {
	log := r.RawLogger
	rolebinding := &rbacv1.RoleBinding{}
	err := r.Get(ctx, types.NamespacedName{Name: pulp.Name, Namespace: pulp.Namespace}, rolebinding)
	expectedRoleBinding := r.pulpRoleBinding(pulp)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new "+pulp.Spec.DeploymentType+" RoleBinding", "Namespace", expectedRoleBinding.Namespace, "Name", expectedRoleBinding.Name)
		r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-API-Ready", "CreatingRoleBinding", "Creating "+pulp.Name+" RoleBinding resource")
		err = r.Create(ctx, expectedRoleBinding)
		if err != nil {
			log.Error(err, "Failed to create new "+pulp.Spec.DeploymentType+" ServiceAccount", "Namespace", expectedRoleBinding.Namespace, "Name", expectedRoleBinding.Name)
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-API-Ready", "ErrorCreatingRoleBinding", "Failed to create "+pulp.Name+" RoleBinding: "+err.Error())
			r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to create new "+pulp.Spec.DeploymentType+" RoleBinding")
			return ctrl.Result{}, err
		}
		// RoleBinding created successfully - return and requeue
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Created", pulp.Spec.DeploymentType+" RoleBinding created")
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get "+pulp.Spec.DeploymentType+" RoleBinding")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *PulpReconciler) pulpSA(m *repomanagerv1alpha1.Pulp) *corev1.ServiceAccount {
	var imagePullSecrets []corev1.LocalObjectReference
	for _, pullSecret := range m.Spec.ImagePullSecrets {
		imagePullSecrets = append(imagePullSecrets, corev1.LocalObjectReference{Name: pullSecret})
	}
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":    m.Name + "-sa",
				"app.kubernetes.io/part-of": m.Spec.DeploymentType,
			},
		},
		ImagePullSecrets: imagePullSecrets,
	}
}

func (r *PulpReconciler) pulpRole(m *repomanagerv1alpha1.Pulp) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":    m.Name + "-role",
				"app.kubernetes.io/part-of": m.Spec.DeploymentType,
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

func (r *PulpReconciler) pulpRoleBinding(m *repomanagerv1alpha1.Pulp) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":    m.Name + "-rolebinding",
				"app.kubernetes.io/part-of": m.Spec.DeploymentType,
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: m.Name,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     m.Name,
		},
	}
}

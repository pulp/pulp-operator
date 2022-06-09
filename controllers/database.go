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
	"reflect"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/util/intstr"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
	"github.com/go-logr/logr"
)

func (r *PulpReconciler) databaseController(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, log logr.Logger) (ctrl.Result, error) {

	// DEPLOYMENT
	found := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-database", Namespace: pulp.Namespace}, found)

	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		dep := r.deploymentForDatabase(pulp)
		log.Info("Creating a new Database Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "Failed to create new Database Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Database Deployment")
		return ctrl.Result{}, err
	}

	// Ensure the deployment size is the same as the spec
	size := pulp.Spec.Api.Replicas
	if *found.Spec.Replicas != size {
		log.Info("Reconciling Database Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
		found.Spec.Replicas = &size
		err = r.Update(ctx, found)
		if err != nil {
			log.Error(err, "Failed to update Database Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return ctrl.Result{}, err
		}
		// Ask to requeue after 1 minute in order to give enough time for the
		// pods be created on the cluster side and the operand be able
		// to do the next update step accurately.
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	// SERVICE
	dbSvc := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-database-svc", Namespace: pulp.Namespace}, dbSvc)

	if err != nil && errors.IsNotFound(err) {
		// Define a new service
		svc := r.serviceForDatabase(pulp)
		log.Info("Creating a new Database Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
		err = r.Create(ctx, svc)
		if err != nil {
			log.Error(err, "Failed to create new Database Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
			return ctrl.Result{}, err
		}
		// Service created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Database Service")
		return ctrl.Result{}, err
	}

	// Ensure the service spec is as expected
	expected_spec := serviceSpec(pulp.Name)

	if !reflect.DeepEqual(expected_spec, dbSvc.Spec) {
		log.Info("The Database service has been modified! Reconciling ...")
		err = r.Update(ctx, serviceObject(pulp.Name, pulp.Namespace))
		if err != nil {
			log.Error(err, "Error trying to update the Database Service object ... ")
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

// deploymentForDatabase returns a postgresql Deployment object
func (r *PulpReconciler) deploymentForDatabase(m *repomanagerv1alpha1.Pulp) *appsv1.Deployment {
	ls := labelsForDatabase(m.Name)
	replicas := m.Spec.Database.Replicas

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-database",
			Namespace: m.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "postgres:13",
						Name:  "postgres",
						Env: []corev1.EnvVar{
							{
								Name:  "POSTGRESQL_DATABASE",
								Value: "pulp",
							},
							{
								Name:  "POSTGRESQL_USER",
								Value: "admin",
							},
							{
								Name:  "POSTGRESQL_PASSWORD",
								Value: "password",
							},
							{
								Name:  "POSTGRES_DB",
								Value: "pulp",
							},
							{
								Name:  "POSTGRES_USER",
								Value: "admin",
							},
							{
								Name:  "POSTGRES_PASSWORD",
								Value: "password",
							},
						},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 5432,
							Name:          "postgres",
						}},
					}},
				},
			},
		},
	}
	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

// labelsForDatabase returns the labels for selecting the resources
// belonging to the given pulp CR name.
func labelsForDatabase(name string) map[string]string {
	return map[string]string{"app": "postgresql", "pulp_cr": name}
}

// serviceForDatabase returns a service object for postgres pods
func (r *PulpReconciler) serviceForDatabase(m *repomanagerv1alpha1.Pulp) *corev1.Service {

	svc := serviceObject(m.Name, m.Namespace)

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(m, svc, r.Scheme)
	return svc
}

func serviceObject(name, namespace string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-database-svc",
			Namespace: namespace,
		},
		Spec: serviceSpec(name),
	}
}

// database service spec
// [TO-DO]
// - the other services will probably be the same as this one with only other
//   names, ports and selectors. This function could be modified to address all pulp
//   service configurations (maybe just adding the corev1.ServicePort and a dictionary
//   for the selectors as function parameters would be enough).
func serviceSpec(name string) corev1.ServiceSpec {

	var serviceInternalTrafficPolicyCluster corev1.ServiceInternalTrafficPolicyType
	serviceInternalTrafficPolicyCluster = "Cluster"

	var ipFamilyPolicyType corev1.IPFamilyPolicyType
	ipFamilyPolicyType = "SingleStack"

	var serviceAffinity corev1.ServiceAffinity
	serviceAffinity = "None"

	var servicePortProto corev1.Protocol
	servicePortProto = "TCP"

	targetPort := intstr.IntOrString{IntVal: 5432}

	var serviceType corev1.ServiceType
	serviceType = "ClusterIP"

	return corev1.ServiceSpec{
		ClusterIP:             "None",
		ClusterIPs:            []string{"None"},
		InternalTrafficPolicy: &serviceInternalTrafficPolicyCluster,
		IPFamilies:            []corev1.IPFamily{"IPv4"},
		IPFamilyPolicy:        &ipFamilyPolicyType,
		Ports: []corev1.ServicePort{{
			Port:       5432,
			Protocol:   servicePortProto,
			TargetPort: targetPort,
		}},
		Selector: map[string]string{
			"app":     "postgresql",
			"pulp_cr": name,
		},
		SessionAffinity: serviceAffinity,
		Type:            serviceType,
	}
}

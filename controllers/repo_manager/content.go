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

	"github.com/go-logr/logr"
	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	"github.com/pulp/pulp-operator/controllers"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ContentResource has the definition and function to provision content objects
type ContentResource struct {
	Definition ResourceDefinition
	Function   func(controllers.FunctionResources) client.Object
}

func (r *RepoManagerReconciler) pulpContentController(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp, log logr.Logger) (ctrl.Result, error) {

	// conditionType is used to update .status.conditions with the current resource state
	conditionType := cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Content-Ready"
	funcResources := controllers.FunctionResources{Context: ctx, Client: r.Client, Pulp: pulp, Scheme: r.Scheme, Logger: log}

	// define the k8s Deployment function based on k8s distribution and deployment type
	deploymentForPulpContent := initDeployment(CONTENT_DEPLOYMENT).Deploy

	// list of pulp-content resources that should be provisioned
	resources := []ContentResource{
		// pulp-content deployment
		{ResourceDefinition{ctx, &appsv1.Deployment{}, pulp.Name + "-content", "Content", conditionType, pulp}, deploymentForPulpContent},
		// pulp-content-svc service
		{ResourceDefinition{ctx, &corev1.Service{}, pulp.Name + "-content-svc", "Content", conditionType, pulp}, serviceForContent},
	}

	// create pulp-content resources
	for _, resource := range resources {
		requeue, err := r.createPulpResource(resource.Definition, resource.Function)
		if err != nil {
			return ctrl.Result{}, err
		} else if requeue {
			return ctrl.Result{Requeue: true}, nil
		}
	}

	// Reconcile Deployment
	deployment := &appsv1.Deployment{}
	r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-content", Namespace: pulp.Namespace}, deployment)
	expected := deploymentForPulpContent(funcResources)
	if requeue, err := controllers.ReconcileObject(funcResources, expected, deployment, conditionType); err != nil || requeue {
		return ctrl.Result{Requeue: requeue}, err
	}

	// Reconcile Service
	cntSvc := &corev1.Service{}
	r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-content-svc", Namespace: pulp.Namespace}, cntSvc)
	newCntSvc := serviceForContent(funcResources)
	if requeue, err := controllers.ReconcileObject(funcResources, newCntSvc, cntSvc, conditionType); err != nil || requeue {
		return ctrl.Result{Requeue: requeue}, err
	}

	return ctrl.Result{}, nil
}

// serviceForContent returns a service object for pulp-content
func serviceForContent(resources controllers.FunctionResources) client.Object {

	pulp := resources.Pulp
	svc := serviceContentObject(pulp.Name, pulp.Namespace, pulp.Spec.DeploymentType)

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(pulp, svc, resources.Scheme)
	return svc
}

func serviceContentObject(name, namespace, deployment_type string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-content-svc",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       deployment_type + "-content",
				"app.kubernetes.io/instance":   deployment_type + "-content-" + name,
				"app.kubernetes.io/component":  "content",
				"app.kubernetes.io/part-of":    deployment_type,
				"app.kubernetes.io/managed-by": deployment_type + "-operator",
				"app":                          "pulp-content",
				"pulp_cr":                      name,
			},
		},
		Spec: serviceContentSpec(name, namespace, deployment_type),
	}
}

// content service spec
func serviceContentSpec(name, namespace, deployment_type string) corev1.ServiceSpec {

	serviceInternalTrafficPolicyCluster := corev1.ServiceInternalTrafficPolicyType("Cluster")
	ipFamilyPolicyType := corev1.IPFamilyPolicyType("SingleStack")
	serviceAffinity := corev1.ServiceAffinity("None")
	servicePortProto := corev1.Protocol("TCP")
	targetPort := intstr.IntOrString{IntVal: 24816}
	serviceType := corev1.ServiceType("ClusterIP")

	return corev1.ServiceSpec{
		InternalTrafficPolicy: &serviceInternalTrafficPolicyCluster,
		IPFamilies:            []corev1.IPFamily{"IPv4"},
		IPFamilyPolicy:        &ipFamilyPolicyType,
		Ports: []corev1.ServicePort{{
			Name:       "content-24816",
			Port:       24816,
			Protocol:   servicePortProto,
			TargetPort: targetPort,
		}},
		Selector: map[string]string{
			"app.kubernetes.io/name":       deployment_type + "-content",
			"app.kubernetes.io/instance":   deployment_type + "-content-" + name,
			"app.kubernetes.io/component":  "content",
			"app.kubernetes.io/part-of":    deployment_type,
			"app.kubernetes.io/managed-by": deployment_type + "-operator",
			"app":                          "pulp-content",
			"pulp_cr":                      name,
		},
		SessionAffinity:          serviceAffinity,
		Type:                     serviceType,
		PublishNotReadyAddresses: true,
	}
}

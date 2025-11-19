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
	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	"github.com/pulp/pulp-operator/controllers"
	"github.com/pulp/pulp-operator/controllers/settings"
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

func (r *RepoManagerReconciler) pulpContentController(ctx context.Context, pulp *pulpv1.Pulp, log logr.Logger) (ctrl.Result, error) {

	// conditionType is used to update .status.conditions with the current resource state
	conditionType := "Pulp-Content-Ready"
	funcResources := controllers.FunctionResources{Context: ctx, Client: r.Client, Pulp: pulp, Scheme: r.Scheme, Logger: log}

	// define the k8s Deployment function based on k8s distribution and deployment type
	deploymentForPulpContent := initDeployment(CONTENT_DEPLOYMENT).Deploy

	deploymentName := settings.CONTENT.DeploymentName(pulp.Name)
	serviceName := settings.ContentService(pulp.Name)

	// list of pulp-content resources that should be provisioned
	resources := []ContentResource{
		// pulp-content deployment
		{ResourceDefinition{ctx, &appsv1.Deployment{}, deploymentName, "Content", conditionType, pulp}, deploymentForPulpContent},
		// pulp-content-svc service
		{ResourceDefinition{ctx, &corev1.Service{}, serviceName, "Content", conditionType, pulp}, serviceForContent},
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
	r.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: pulp.Namespace}, deployment)
	expected := deploymentForPulpContent(funcResources)
	if requeue, err := controllers.ReconcileObject(funcResources, expected, deployment, conditionType, controllers.PulpDeployment{}); err != nil || requeue {
		return ctrl.Result{Requeue: requeue}, err
	}

	// Reconcile Service
	cntSvc := &corev1.Service{}
	r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: pulp.Namespace}, cntSvc)
	newCntSvc := serviceForContent(funcResources)
	if requeue, err := controllers.ReconcileObject(funcResources, newCntSvc, cntSvc, conditionType, controllers.PulpService{}); err != nil || requeue {
		return ctrl.Result{Requeue: requeue}, err
	}

	return ctrl.Result{}, nil
}

// serviceForContent returns a service object for pulp-content
func serviceForContent(resources controllers.FunctionResources) client.Object {

	pulp := resources.Pulp
	svc := serviceContentObject(*pulp)

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(pulp, svc, resources.Scheme)
	return svc
}

func serviceContentObject(pulp pulpv1.Pulp) *corev1.Service {
	name := pulp.Name
	namespace := pulp.Namespace
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      settings.ContentService(name),
			Namespace: namespace,
			Labels:    settings.PulpcoreLabels(pulp, settings.CONTENT),
		},
		Spec: serviceContentSpec(pulp),
	}
}

// content service spec
func serviceContentSpec(pulp pulpv1.Pulp) corev1.ServiceSpec {

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
		Selector:        settings.PulpcoreLabels(pulp, settings.CONTENT),
		SessionAffinity: serviceAffinity,
		Type:            serviceType,
	}
}

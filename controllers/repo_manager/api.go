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
	"github.com/pulp/pulp-operator/controllers/settings"
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

// ApiResource has the definition and function to provision api objects
type ApiResource struct {
	Definition ResourceDefinition
	Function   func(controllers.FunctionResources) client.Object
}

// pulpApiController provision and reconciles api objects
func (r *RepoManagerReconciler) pulpApiController(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp, log logr.Logger) (ctrl.Result, error) {

	// conditionType is used to update .status.conditions with the current resource state
	conditionType := cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-API-Ready"
	funcResources := controllers.FunctionResources{Context: ctx, Client: r.Client, Pulp: pulp, Scheme: r.Scheme, Logger: log}

	// define the k8s Deployment function based on k8s distribution and deployment type
	deploymentForPulpApi := initDeployment(API_DEPLOYMENT).Deploy

	deploymentName := settings.API.DeploymentName(pulp.Name)
	serviceName := settings.ApiService(pulp.Name)

	// list of pulp-api resources that should be provisioned
	resources := []ApiResource{
		// pulp-api deployment
		{ResourceDefinition{ctx, &appsv1.Deployment{}, deploymentName, "Api", conditionType, pulp}, deploymentForPulpApi},
		// pulp-api-svc service
		{ResourceDefinition{ctx, &corev1.Service{}, serviceName, "Api", conditionType, pulp}, serviceForAPI},
	}

	// create telemetry resources
	if pulp.Spec.Telemetry.Enabled {
		telemetry := []ApiResource{
			{ResourceDefinition{ctx, &corev1.ConfigMap{}, settings.OtelConfigMapName(pulp.Name), "Telemetry", conditionType, pulp}, controllers.OtelConfigMap},
			{ResourceDefinition{ctx, &corev1.Service{}, settings.OtelServiceName(pulp.Name), "Telemetry", conditionType, pulp}, controllers.ServiceOtel},
		}
		resources = append(resources, telemetry...)
	}

	// create pulp-api resources
	for _, resource := range resources {
		requeue, err := r.createPulpResource(resource.Definition, resource.Function)
		if err != nil {
			return ctrl.Result{}, err
		} else if requeue {
			return ctrl.Result{Requeue: true}, nil
		}
	}

	// Ensure the deployment spec is as expected
	apiDeployment := &appsv1.Deployment{}
	r.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: pulp.Namespace}, apiDeployment)
	expected := deploymentForPulpApi(funcResources)
	if requeue, err := controllers.ReconcileObject(funcResources, expected, apiDeployment, conditionType, controllers.PulpDeployment{}); err != nil || requeue {
		return ctrl.Result{Requeue: requeue}, err
	}

	// Ensure the service spec is as expected
	apiSvc := &corev1.Service{}
	r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: pulp.Namespace}, apiSvc)
	expectedSvc := serviceForAPI(funcResources)
	if requeue, err := controllers.ReconcileObject(funcResources, expectedSvc, apiSvc, conditionType, controllers.PulpService{}); err != nil || requeue {
		return ctrl.Result{Requeue: requeue}, err
	}

	// telemetry resources reconciliation
	if pulp.Spec.Telemetry.Enabled {
		// Ensure otelConfigMap is as expected
		telemetryConfigMap := &corev1.ConfigMap{}
		r.Get(ctx, types.NamespacedName{Name: settings.OtelConfigMapName(pulp.Name), Namespace: pulp.Namespace}, telemetryConfigMap)
		expectedTelemetryConfigMap := controllers.OtelConfigMap(funcResources)
		if requeue, err := controllers.ReconcileObject(funcResources, expectedTelemetryConfigMap, telemetryConfigMap, conditionType, controllers.PulpConfigMap{}); err != nil || requeue {
			return ctrl.Result{Requeue: requeue}, err
		}

		// Ensure otelService is as expected
		telemetryService := &corev1.Service{}
		r.Get(ctx, types.NamespacedName{Name: settings.OtelServiceName(pulp.Name), Namespace: pulp.Namespace}, telemetryService)
		expectedTelemetryService := controllers.ServiceOtel(funcResources)
		if requeue, err := controllers.ReconcileObject(funcResources, expectedTelemetryService, telemetryService, conditionType, controllers.PulpService{}); err != nil || requeue {
			return ctrl.Result{Requeue: requeue}, err
		}
	}

	return ctrl.Result{}, nil
}

// serviceForAPI returns a service object for pulp-api
func serviceForAPI(resources controllers.FunctionResources) client.Object {
	pulp := resources.Pulp
	svc := serviceAPIObject(*pulp)

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(pulp, svc, resources.Scheme)
	return svc
}

func serviceAPIObject(pulp repomanagerpulpprojectorgv1beta2.Pulp) *corev1.Service {
	name := pulp.Name
	namespace := pulp.Namespace

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      settings.ApiService(name),
			Namespace: namespace,
			Labels:    settings.PulpcoreLabels(pulp, "api"),
		},
		Spec: serviceAPISpec(pulp),
	}
}

// api service spec
func serviceAPISpec(pulp repomanagerpulpprojectorgv1beta2.Pulp) corev1.ServiceSpec {

	serviceInternalTrafficPolicyCluster := corev1.ServiceInternalTrafficPolicyType("Cluster")
	ipFamilyPolicyType := corev1.IPFamilyPolicyType("SingleStack")
	serviceAffinity := corev1.ServiceAffinity("None")
	servicePortProto := corev1.Protocol("TCP")
	targetPort := intstr.IntOrString{IntVal: 24817}
	serviceType := corev1.ServiceType("ClusterIP")

	return corev1.ServiceSpec{
		InternalTrafficPolicy: &serviceInternalTrafficPolicyCluster,
		IPFamilies:            []corev1.IPFamily{"IPv4"},
		IPFamilyPolicy:        &ipFamilyPolicyType,
		Ports: []corev1.ServicePort{{
			Name:       "api-24817",
			Port:       24817,
			Protocol:   servicePortProto,
			TargetPort: targetPort,
		}},
		Selector:        settings.PulpcoreLabels(pulp, "api"),
		SessionAffinity: serviceAffinity,
		Type:            serviceType,
	}
}

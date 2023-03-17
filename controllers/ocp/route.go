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

package ocp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/pulp/pulp-operator/controllers"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// statusReturn is used to control goroutines execution
type statusReturn struct {
	ctrl.Result
	error
}

// RoutePlugin defines a plugin route.
type RoutePlugin struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	ServiceName string `json:"serviceName"`
	TargetPort  string `json:"targetPort"`
	Rewrite     string `json:"rewrite"`
}

// PodExec contains the configs to execute a command inside a pod
type PodExec struct {
	RESTClient rest.Interface
	RESTConfig *rest.Config
	Scheme     *runtime.Scheme
}

// PulpRouteController creates the routes based on snippets defined in pulp-worker pod
func PulpRouteController(resources controllers.FunctionResources, restClient rest.Interface, restConfig *rest.Config) (ctrl.Result, error) {

	pulp := resources.Pulp
	log := resources.Logger
	ctx := resources.Context

	// conditionType is used to update .status.conditions with the current resource state
	conditionType := cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Route-Ready"

	podList := &corev1.PodList{}
	labels := map[string]string{
		"app.kubernetes.io/part-of":    pulp.Spec.DeploymentType,
		"app.kubernetes.io/managed-by": pulp.Spec.DeploymentType + "-operator",
		"app.kubernetes.io/instance":   pulp.Spec.DeploymentType + "-worker-" + pulp.Name,
		"app.kubernetes.io/component":  "worker",
	}
	listOpts := []client.ListOption{
		client.InNamespace(pulp.Namespace),
		client.MatchingLabels(labels),
	}
	if err := resources.Client.List(ctx, podList, listOpts...); err != nil {
		log.Error(err, "Failed to list Worker pods", "Pulp.Namespace", pulp.Namespace, "Pulp.Name", pulp.Name)
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}
	var isPodRunning bool = false
	var pod = corev1.Pod{}
	for _, p := range podList.Items {
		log.V(1).Info("Checking Worker pod", "Pod", p.Name, "Status", p.Status.Phase)
		if p.Status.Phase == "Running" {
			log.V(1).Info("Running!", "Pod", p.Name, "Status", p.Status.Phase)
			isPodRunning = true
			pod = p
			break
		} else {
			log.Info("Worker Pod isn't running yet!", "Pod", p.Name, "Status", p.Status.Phase)
		}
	}

	if !isPodRunning {
		log.Info("Worker pod isn't running yet!")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}
	execCmd := []string{
		"/usr/bin/route_paths.py", pulp.Name,
	}
	cmdOutput, err := controllers.ContainerExec(PodExec{restClient, restConfig, resources.Scheme}, &pod, execCmd, "worker", pod.Namespace)
	if err != nil {
		controllers.CustomZapLogger().Warn(err.Error() + " Failed to get routes from " + pod.Name)
		controllers.UpdateStatus(ctx, resources.Client, pulp, metav1.ConditionFalse, conditionType, "Failed to get routes!", "FailedGet"+pod.Name)
		return ctrl.Result{Requeue: true}, nil
	}
	var pulpPlugins []RoutePlugin
	json.Unmarshal([]byte(cmdOutput), &pulpPlugins)
	defaultPlugins := []RoutePlugin{
		{
			Name:        pulp.Name + "-content",
			Path:        controllers.GetPulpSetting(pulp, "content_path_prefix"),
			TargetPort:  "content-24816",
			ServiceName: pulp.Name + "-content-svc",
		},
		{
			Name:        pulp.Name + "-api-v3",
			Path:        controllers.GetPulpSetting(pulp, "api_root") + "api/v3/",
			TargetPort:  "api-24817",
			ServiceName: pulp.Name + "-api-svc",
		},
		{
			Name:        pulp.Name + "-auth",
			Path:        "/auth/login/",
			TargetPort:  "api-24817",
			ServiceName: pulp.Name + "-api-svc",
		},
		{
			Name:        pulp.Name,
			Path:        "/",
			TargetPort:  "api-24817",
			ServiceName: pulp.Name + "-api-svc",
		},
	}
	routeHost := GetRouteHost(ctx, resources.Client, pulp)
	pulpPlugins = append(defaultPlugins, pulpPlugins...)

	// channel used to receive the return value from each goroutine
	c := make(chan statusReturn)

	for _, plugin := range pulpPlugins {

		// provision each route resource concurrently
		go func(plugin RoutePlugin) {

			// get route
			currentRoute := &routev1.Route{}
			resources := controllers.FunctionResources{Context: ctx, Client: resources.Client, Pulp: pulp, Scheme: resources.Scheme, Logger: log}

			expectedRoute := PulpRouteObject(ctx, resources, &plugin, routeHost)
			err := resources.Client.Get(ctx, types.NamespacedName{Name: plugin.Name, Namespace: pulp.Namespace}, currentRoute)

			// Create the route in case it is not found
			if err != nil && errors.IsNotFound(err) {
				log.Info("Creating a new route", "Route.Namespace", expectedRoute.Namespace, "Route.Name", expectedRoute.Name)
				controllers.UpdateStatus(ctx, resources.Client, pulp, metav1.ConditionFalse, conditionType, "CreatingRoute", "Creating "+pulp.Name+"-route")
				if err := resources.Client.Create(ctx, expectedRoute); err != nil {
					log.Error(err, "Failed to create new route", "Route.Namespace", expectedRoute.Namespace, "Route.Name", expectedRoute.Name)
					controllers.UpdateStatus(ctx, resources.Client, pulp, metav1.ConditionFalse, conditionType, "ErrorCreatingRoute", "Failed to create "+pulp.Name+"-route: "+err.Error())
					c <- statusReturn{ctrl.Result{}, err}
					return
				}
				c <- statusReturn{ctrl.Result{}, nil}
				return
			} else if err != nil {
				log.Error(err, "Failed to get route")
				c <- statusReturn{ctrl.Result{}, err}
				return
			}

			// Ensure route specs are as expected
			if requeue, err := controllers.ReconcileObject(resources, expectedRoute, currentRoute, conditionType); err != nil || requeue {
				c <- statusReturn{ctrl.Result{Requeue: requeue}, err}
				return
			}

			// Ensure route labels and annotations are as expected
			if requeue, err := controllers.ReconcileMetadata(resources, expectedRoute, currentRoute, conditionType); err != nil || requeue {
				c <- statusReturn{ctrl.Result{Requeue: requeue}, err}
				return
			}

		}(plugin)

		// if there is no element in chan it means the goroutine didnt have any errors
		// nor any reconciliation loop (ctrl.Result{}, nil) requested
		// we need to check this to avoid getting into a blocked state with the channel
		// waiting for a value to be "consumed" which would never be delivered
		if len(c) > 0 {
			pluginRoutineReturn := <-c
			return pluginRoutineReturn.Result, pluginRoutineReturn.error
		}
	}

	// we should only update the status when Route-Ready==false
	if v1.IsStatusConditionFalse(pulp.Status.Conditions, conditionType) {
		controllers.UpdateStatus(ctx, resources.Client, pulp, metav1.ConditionTrue, conditionType, "RouteTasksFinished", "All Route tasks ran successfully")
	}
	return ctrl.Result{}, nil
}

// PulpRouteObject returns the route object with the specs defined in pulp CR
func PulpRouteObject(ctx context.Context, resources controllers.FunctionResources, p *RoutePlugin, routeHost string) *routev1.Route {

	log := logr.Logger{}
	weight := int32(100)

	// set HAProxy default values
	hAProxyTimeout := resources.Pulp.Spec.HAProxyTimeout
	if len(hAProxyTimeout) == 0 {
		hAProxyTimeout = "180s"
	}
	annotation := map[string]string{
		"haproxy.router.openshift.io/timeout": hAProxyTimeout,
	}

	if len(p.Rewrite) > 0 {
		annotation["haproxy.router.openshift.io/rewrite-target"] = p.Rewrite
	}

	labels := map[string]string{}
	labels["pulp_cr"] = resources.Pulp.Name
	labels["owner"] = "pulp-dev"
	for k, v := range resources.Pulp.Spec.RouteLabels {
		labels[k] = v
	}
	for key, val := range resources.Pulp.Spec.RouteAnnotations {
		annotation[key] = val
	}

	certTLSConfig := routev1.TLSConfig{}
	if len(resources.Pulp.Spec.RouteTLSSecret) > 0 {
		certData, err := controllers.RetrieveSecretData(ctx, resources.Pulp.Spec.RouteTLSSecret, resources.Pulp.Namespace, true, resources.Client, "key", "certificate")
		if err != nil {
			log.Error(err, "Failed to retrieve secret data.")
		} else {
			certTLSConfig.Certificate = certData["certificate"]
			certTLSConfig.Key = certData["key"]

			// caCertificate is optional
			certData, _ = controllers.RetrieveSecretData(ctx, resources.Pulp.Spec.RouteTLSSecret, resources.Pulp.Namespace, false, resources.Client, "caCertificate")
			certTLSConfig.CACertificate = certData["caCertificate"]
		}
	}

	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:        p.Name,
			Namespace:   resources.Pulp.Namespace,
			Annotations: annotation,
			Labels:      labels,
		},
		Spec: routev1.RouteSpec{
			Host: routeHost,
			Path: p.Path,
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString(p.TargetPort),
			},
			TLS: &routev1.TLSConfig{
				Termination:                   routev1.TLSTerminationEdge,
				InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
				Certificate:                   certTLSConfig.Certificate,
				Key:                           certTLSConfig.Key,
				CACertificate:                 certTLSConfig.CACertificate,
			},
			To: routev1.RouteTargetReference{
				Kind:   "Service",
				Name:   p.ServiceName,
				Weight: &weight,
			},
			WildcardPolicy: routev1.WildcardPolicyNone,
		},
	}

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(resources.Pulp, route, resources.Scheme)
	return route
}

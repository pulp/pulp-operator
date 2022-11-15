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
	"encoding/json"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	routev1 "github.com/openshift/api/route/v1"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	repomanagerv1alpha1 "github.com/pulp/pulp-operator/api/v1alpha1"
	"github.com/pulp/pulp-operator/controllers"
)

// pulpRouteController creates the routes based on snippets defined in pulp-worker pod
func (r *RepoManagerReconciler) pulpRouteController(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, log logr.Logger) (ctrl.Result, error) {

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
	if err := r.List(ctx, podList, listOpts...); err != nil {
		log.Error(err, "Failed to list Worker pods", "Pulp.Namespace", pulp.Namespace, "Pulp.Name", pulp.Name)
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}
	var IsPodRunning bool = false
	var pod = corev1.Pod{}
	for _, p := range podList.Items {
		log.V(1).Info("Checking Worker pod", "Pod", p.Name, "Status", p.Status.Phase)
		if p.Status.Phase == "Running" {
			log.V(1).Info("Running!", "Pod", p.Name, "Status", p.Status.Phase)
			IsPodRunning = true
			pod = p
			break
		} else {
			log.Info("Worker Pod isn't running yet!", "Pod", p.Name, "Status", p.Status.Phase)
		}
	}

	if !IsPodRunning {
		log.Info("Worker pod isn't running yet!")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}
	execCmd := []string{
		"/usr/bin/route_paths.py", pulp.Name,
	}
	cmdOutput, err := controllers.ContainerExec(r, &pod, execCmd, "worker", pod.Namespace)
	if err != nil {
		controllers.CustomZapLogger().Warn(err.Error() + " Failed to get routes from " + pod.Name)
		r.updateStatus(ctx, pulp, metav1.ConditionFalse, conditionType, "Failed to get routes!", "FailedGet"+pod.Name)
		return ctrl.Result{Requeue: true}, nil
	}
	var pulpPlugins []RoutePlugin
	json.Unmarshal([]byte(cmdOutput), &pulpPlugins)
	defaultPlugins := []RoutePlugin{
		{
			Name:        pulp.Name + "-content",
			Path:        getPulpSetting(pulp, "content_path_prefix"),
			TargetPort:  "content-24816",
			ServiceName: pulp.Name + "-content-svc",
		},
		{
			Name:        pulp.Name + "-api-v3",
			Path:        getPulpSetting(pulp, "api_root") + "api/v3/",
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
	routeHost := pulp.Spec.RouteHost
	if len(pulp.Spec.RouteHost) == 0 {
		ingress := &configv1.Ingress{}
		r.Get(ctx, types.NamespacedName{Name: "cluster"}, ingress)
		routeHost = pulp.Name + "." + ingress.Spec.Domain
	}
	pulpPlugins = append(defaultPlugins, pulpPlugins...)

	// channel used to receive the return value from each goroutine
	c := make(chan statusReturn)

	for _, plugin := range pulpPlugins {

		// provision each route resource concurrently
		go func(plugin RoutePlugin) {

			// get route
			currentRoute := &routev1.Route{}
			expectedRoute := r.pulpRouteObject(pulp, &plugin, routeHost)
			err := r.Get(ctx, types.NamespacedName{Name: plugin.Name, Namespace: pulp.Namespace}, currentRoute)

			// Create the route in case it is not found
			if err != nil && errors.IsNotFound(err) {
				log.Info("Creating a new route", "Route.Namespace", expectedRoute.Namespace, "Route.Name", expectedRoute.Name)
				r.updateStatus(ctx, pulp, metav1.ConditionFalse, conditionType, "CreatingRoute", "Creating "+pulp.Name+"-route")
				if err := r.Create(ctx, expectedRoute); err != nil {
					log.Error(err, "Failed to create new route", "Route.Namespace", expectedRoute.Namespace, "Route.Name", expectedRoute.Name)
					r.updateStatus(ctx, pulp, metav1.ConditionFalse, conditionType, "ErrorCreatingRoute", "Failed to create "+pulp.Name+"-route: "+err.Error())
					r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to create new route")
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
			if err := r.reconcileObject(ctx, pulp, expectedRoute, currentRoute, conditionType, log); err != nil {
				log.Error(err, "Failed to update route spec")
				c <- statusReturn{ctrl.Result{}, err}
				return
			}

			// Ensure route labels and annotations are as expected
			if err := r.reconcileMetadata(ctx, pulp, expectedRoute, currentRoute, conditionType, log); err != nil {
				log.Error(err, "Failed to update route labels")
				c <- statusReturn{ctrl.Result{}, err}
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
		r.updateStatus(ctx, pulp, metav1.ConditionTrue, conditionType, "RouteTasksFinished", "All Route tasks ran successfully")
		r.recorder.Event(pulp, corev1.EventTypeNormal, "RouteReady", "All Route tasks ran successfully")
	}
	return ctrl.Result{}, nil
}

// pulpRouteObject returns the route object with the specs defined in pulp CR
func (r *RepoManagerReconciler) pulpRouteObject(m *repomanagerv1alpha1.Pulp, p *RoutePlugin, routeHost string) *routev1.Route {

	weight := int32(100)

	// set HAProxy default values
	hAProxyTimeout := m.Spec.HAProxyTimeout
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
	labels["pulp_cr"] = m.Name
	labels["owner"] = "pulp-dev"
	for k, v := range m.Spec.RouteLabels {
		labels[k] = v
	}
	for key, val := range m.Spec.RouteAnnotations {
		annotation[key] = val
	}

	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:        p.Name,
			Namespace:   m.Namespace,
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
	ctrl.SetControllerReference(m, route, r.Scheme)
	return route
}

// RoutePlugin defines a plugin route.
type RoutePlugin struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	ServiceName string `json:"serviceName"`
	TargetPort  string `json:"targetPort"`
	Rewrite     string `json:"rewrite"`
}

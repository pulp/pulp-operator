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
	"encoding/json"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	repomanagerv1alpha1 "github.com/pulp/pulp-operator/api/v1alpha1"
	"github.com/pulp/pulp-operator/controllers"
)

func (r *PulpReconciler) pulpRouteController(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, log logr.Logger) (ctrl.Result, error) {

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
		log.Info("Checking Worker pod", "Pod", p.Name, "Status", p.Status.Phase)
		if p.Status.Phase == "Running" {
			log.Info("Running!", "Pod", p.Name, "Status", p.Status.Phase)
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
		log.Error(err, "Failed to get routes from "+pod.Name)
		r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-Route-Ready", "Failed to get routes!", "FailedGet"+pod.Name)
		return ctrl.Result{}, err
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
	for _, plugin := range pulpPlugins {
		// get route
		pulpRoute := &routev1.Route{}
		err := r.Get(ctx, types.NamespacedName{Name: plugin.Name, Namespace: pulp.Namespace}, pulpRoute)

		// Create the route in case it is not found
		if err != nil && errors.IsNotFound(err) {
			routePwd := pulpRouteObject(pulp, &plugin, routeHost)
			ctrl.SetControllerReference(pulp, routePwd, r.Scheme)
			log.Info("Creating a new route", "Route.Namespace", routePwd.Namespace, "Route.Name", routePwd.Name)
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-Route-Ready", "CreatingRoute", "Creating "+pulp.Name+"-route")
			err = r.Create(ctx, routePwd)
			if err != nil {
				log.Error(err, "Failed to create new route", "Route.Namespace", routePwd.Namespace, "Route.Name", routePwd.Name)
				r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-Route-Ready", "ErrorCreatingRoute", "Failed to create "+pulp.Name+"-route: "+err.Error())
				r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to create new route")
				return ctrl.Result{}, err
			}
		} else if err != nil {
			log.Error(err, "Failed to get route")
			return ctrl.Result{}, err
		}
	}
	r.updateStatus(ctx, pulp, metav1.ConditionTrue, pulp.Spec.DeploymentType+"-Route-Ready", "RouteTasksFinished", "All Route tasks ran successfully")
	r.recorder.Event(pulp, corev1.EventTypeNormal, "RouteReady", "All Route tasks ran successfully")
	return ctrl.Result{}, nil
}

// pulp-route
func pulpRouteObject(m *repomanagerv1alpha1.Pulp, p *RoutePlugin, routeHost string) *routev1.Route {
	weight := int32(100)
	annotation := map[string]string{
		"haproxy.router.openshift.io/timeout": m.Spec.HAProxyTimeout,
	}
	if len(p.Rewrite) > 0 {
		annotation["haproxy.router.openshift.io/rewrite-target"] = p.Rewrite
	}
	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:        p.Name,
			Namespace:   m.Namespace,
			Annotations: annotation,
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
}

// RoutePlugin defines a plugin route.
type RoutePlugin struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	ServiceName string `json:"serviceName"`
	TargetPort  string `json:"targetPort"`
	Rewrite     string `json:"rewrite"`
}

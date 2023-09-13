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
	"strings"
	"time"

	"github.com/go-logr/logr"
	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	"github.com/pulp/pulp-operator/controllers"
	pulp_ocp "github.com/pulp/pulp-operator/controllers/ocp"
	"github.com/pulp/pulp-operator/controllers/settings"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *RepoManagerReconciler) pulpIngressController(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp, log logr.Logger) (ctrl.Result, error) {

	// conditionType is used to update .status.conditions with the current resource state
	conditionType := cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Ingress-Ready"

	podList := &corev1.PodList{}
	labels := map[string]string{
		"app.kubernetes.io/part-of":    pulp.Spec.DeploymentType,
		"app.kubernetes.io/managed-by": pulp.Spec.DeploymentType + "-operator",
		"app.kubernetes.io/instance":   pulp.Spec.DeploymentType + "-content-" + pulp.Name,
		"app.kubernetes.io/component":  "content",
	}
	listOpts := []client.ListOption{
		client.InNamespace(pulp.Namespace),
		client.MatchingLabels(labels),
	}
	if err := r.List(ctx, podList, listOpts...); err != nil {
		log.Error(err, "Failed to list Content pods", "Pulp.Namespace", pulp.Namespace, "Pulp.Name", pulp.Name)
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}
	var IsPodRunning bool = false
	var pod = corev1.Pod{}
	for _, p := range podList.Items {
		log.V(1).Info("Checking Content pod", "Pod", p.Name, "Status", p.Status.Phase)
		if p.Status.Phase == "Running" {
			log.V(1).Info("Running!", "Pod", p.Name, "Status", p.Status.Phase)
			IsPodRunning = true
			pod = p
			break
		} else {
			log.Info("Content Pod isn't running yet!", "Pod", p.Name, "Status", p.Status.Phase)
		}
	}

	if !IsPodRunning {
		log.Info("Content pod isn't running yet!")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}
	execCmd := []string{
		"/usr/bin/route_paths.py", pulp.Name,
	}
	cmdOutput, err := controllers.ContainerExec(r, &pod, execCmd, "content", pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to get ingresss from "+pod.Name)
		controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, conditionType, "Failed to get ingresss!", "FailedGet"+pod.Name)
		return ctrl.Result{}, err
	}
	var pulpPlugins []controllers.IngressPlugin
	json.Unmarshal([]byte(cmdOutput), &pulpPlugins)
	defaultPlugins := []controllers.IngressPlugin{
		{
			Name:        pulp.Name + "-content",
			Path:        controllers.GetPulpSetting(pulp, "content_path_prefix"),
			TargetPort:  "content-24816",
			ServiceName: settings.ContentService(pulp.Name),
		},
		{
			Name:        pulp.Name + "-api-v3",
			Path:        controllers.GetPulpSetting(pulp, "api_root") + "api/v3/",
			TargetPort:  "api-24817",
			ServiceName: settings.ApiService(pulp.Name),
		},
		{
			Name:        pulp.Name + "-auth",
			Path:        "/auth/login/",
			TargetPort:  "api-24817",
			ServiceName: settings.ApiService(pulp.Name),
		},
		{
			Name:        pulp.Name,
			Path:        "/",
			TargetPort:  "api-24817",
			ServiceName: settings.ApiService(pulp.Name),
		},
	}
	pulpPlugins = append(defaultPlugins, pulpPlugins...)

	// get ingress
	currentIngress := &netv1.Ingress{}
	resources := controllers.FunctionResources{Context: ctx, Client: r.Client, Pulp: pulp, Scheme: r.Scheme, Logger: log}
	ingress, err := r.initIngress(resources)
	if err != nil {
		return ctrl.Result{}, err
	}
	expectedIngress, err := ingress.Deploy(resources, pulpPlugins)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.Get(ctx, types.NamespacedName{Name: pulp.Name, Namespace: pulp.Namespace}, currentIngress)

	// Create the ingress in case it is not found
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new ingress", "Ingress.Namespace", expectedIngress.Namespace, "Ingress.Name", expectedIngress.Name)
		controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, conditionType, "CreatingIngress", "Creating "+pulp.Name+"-ingress")
		err = r.Create(ctx, expectedIngress)
		if err != nil {
			log.Error(err, "Failed to create new ingress", "Ingress.Namespace", expectedIngress.Namespace, "Ingress.Name", expectedIngress.Name)
			controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, conditionType, "ErrorCreatingIngress", "Failed to create "+pulp.Name+"-ingress: "+err.Error())
			r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to create new ingress")
			return ctrl.Result{}, err
		}
	} else if err != nil {
		log.Error(err, "Failed to get ingress")
		return ctrl.Result{}, err
	}

	// Ensure ingress specs are as expected
	if requeue, err := controllers.ReconcileObject(controllers.FunctionResources{Context: ctx, Client: r.Client, Pulp: pulp, Scheme: r.Scheme, Logger: log}, expectedIngress, currentIngress, conditionType, controllers.PulpIngress{}); err != nil || requeue {
		return ctrl.Result{Requeue: requeue}, err
	}

	// Ensure ingress labels and annotations are as expected
	if requeue, err := controllers.ReconcileMetadata(controllers.FunctionResources{Context: ctx, Client: r.Client, Pulp: pulp, Scheme: r.Scheme, Logger: log}, expectedIngress, currentIngress, conditionType); err != nil || requeue {
		return ctrl.Result{Requeue: requeue}, err
	}

	// we should only update the status when Ingress-Ready==false
	if v1.IsStatusConditionFalse(pulp.Status.Conditions, conditionType) {
		controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionTrue, conditionType, "IngressTasksFinished", "All Ingress tasks ran successfully")
		r.recorder.Event(pulp, corev1.EventTypeNormal, "IngressReady", "All Ingress tasks ran successfully")
	}

	if expectedIngress.Annotations["web"] == "true" {
		log.V(1).Info("Running web tasks")
		pulpController, err := r.pulpWebController(ctx, pulp, log)
		if needsRequeue(err, pulpController) {
			return pulpController, err
		}
	}
	return ctrl.Result{}, nil
}

// IngressObj represents the k8s "Ingress" resource
type IngressObj struct {
	Ingresser
}

// initIngress returns a concrete ingress object based on k8s distribution and
// ingress controller (nginx, haproxy, etc)
func (r *RepoManagerReconciler) initIngress(resources controllers.FunctionResources) (*IngressObj, error) {

	// if the ingressclass provided has nginx as controller set the IngressObj as IngressNginx
	if controllers.IsNginxIngressSupported(resources.Pulp) {
		return &IngressObj{IngressNginx{}}, nil
	}

	// if this is an ocp cluster and ingressclass provided is the ocp default set the IngressObj as IngressOCP
	if isOpenShift, _ := controllers.IsOpenShift(); isOpenShift && resources.Pulp.Spec.IngressClassName == controllers.DefaultOCPIngressClass {
		return &IngressObj{pulp_ocp.IngressOCP{}}, nil
	}

	return &IngressObj{IngressOthers{}}, nil
}

// Ingresser is an interface for the several ingress types/controllers (nginx,haproxy)
type Ingresser interface {
	Deploy(controllers.FunctionResources, []controllers.IngressPlugin) (*netv1.Ingress, error)
}

type IngressNginx struct{}

// Deploy returns an ingress using nginx controller
func (i IngressNginx) Deploy(resources controllers.FunctionResources, plugins []controllers.IngressPlugin) (*netv1.Ingress, error) {

	pulp := resources.Pulp

	ingress, err := controllers.IngressDefaults(resources, plugins)
	if err != nil {
		return nil, err
	}

	annotation := map[string]string{
		"web": "false",
	}
	var paths []netv1.HTTPIngressPath
	var path netv1.HTTPIngressPath
	pathType := netv1.PathTypePrefix
	rewrite := ""

	// set Nginx default values
	nginxProxyBodySize := pulp.Spec.NginxProxyBodySize
	if len(nginxProxyBodySize) == 0 {
		nginxProxyBodySize = "0"
	}
	nginxMaxBodySize := pulp.Spec.NginxMaxBodySize
	if len(nginxMaxBodySize) == 0 {
		nginxMaxBodySize = "10m"
	}
	nginxProxyReadTimeout := pulp.Spec.NginxProxyReadTimeout
	if len(nginxProxyReadTimeout) == 0 {
		nginxProxyReadTimeout = "120s"
	}
	nginxProxySendTimeout := pulp.Spec.NginxProxySendTimeout
	if len(nginxProxySendTimeout) == 0 {
		nginxProxySendTimeout = "120s"
	}
	nginxProxyConnectTimeout := pulp.Spec.NginxProxyConnectTimeout
	if len(nginxProxyConnectTimeout) == 0 {
		nginxProxyConnectTimeout = "120s"
	}

	for _, plugin := range plugins {
		if len(plugin.Rewrite) > 0 {
			rewrite = "rewrite ^" + strings.TrimRight(plugin.Path, "/") + "* " + plugin.Rewrite + ";"
			if strings.Contains(annotation["nginx.ingress.kubernetes.io/configuration-snippet"], rewrite) {
				continue
			}
			annotation["nginx.ingress.kubernetes.io/configuration-snippet"] = rewrite
			continue
		}
		path = netv1.HTTPIngressPath{
			Path:     plugin.Path,
			PathType: &pathType,
			Backend: netv1.IngressBackend{
				Service: &netv1.IngressServiceBackend{
					Name: plugin.ServiceName,
					Port: netv1.ServiceBackendPort{
						Name: plugin.TargetPort,
					},
				},
			},
		}
		paths = append(paths, path)
	}

	annotation["nginx.ingress.kubernetes.io/proxy-body-size"] = nginxProxyBodySize
	annotation["nginx.org/client-max-body-size"] = nginxMaxBodySize
	annotation["nginx.ingress.kubernetes.io/proxy-read-timeout"] = nginxProxyReadTimeout
	annotation["nginx.ingress.kubernetes.io/proxy-connect-timeout"] = nginxProxyConnectTimeout
	annotation["nginx.ingress.kubernetes.io/proxy-send-timeout"] = nginxProxySendTimeout

	ingress.ObjectMeta.Annotations = annotation
	ingress.Spec.IngressClassName = &pulp.Spec.IngressClassName
	ingress.Spec.Rules[0].IngressRuleValue = netv1.IngressRuleValue{
		HTTP: &netv1.HTTPIngressRuleValue{
			Paths: paths,
		},
	}

	return ingress, nil
}

type IngressOthers struct{}

// Deploy returns an ingress with the default configurations
func (i IngressOthers) Deploy(resources controllers.FunctionResources, plugins []controllers.IngressPlugin) (*netv1.Ingress, error) {
	ingress, err := controllers.IngressDefaults(resources, plugins)
	if err != nil {
		return nil, err
	}
	return ingress, nil
}

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
	"github.com/pulp/pulp-operator/controllers"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// IngressOCP is the Ingress definition for OCP clusters
type IngressOCP struct{}

// Deploy returns an Ingress based on default OCP IngressController (openshift.io/ingress-to-route)
func (i IngressOCP) Deploy(resources controllers.FunctionResources, plugins []controllers.IngressPlugin) (*netv1.Ingress, error) {

	// deploy redirect ingress
	if err := deployOCPIngressRedirect(resources, plugins); err != nil {
		return &netv1.Ingress{}, err
	}

	// deploy default ingress
	return deployOCPIngress(resources, plugins)
}

// deployRedir defines the ocp ingress spec with the redirect rules
func deployOCPIngressRedirect(resources controllers.FunctionResources, plugins []controllers.IngressPlugin) error {
	pulp := resources.Pulp
	log := resources.Logger
	ingressName := pulp.Name + "-redirect"
	conditionType := cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Ingress-Ready"

	expectedIngress, err := controllers.IngressDefaults(resources, plugins)
	if err != nil {
		return nil
	}

	redirectAnnotation := map[string]string{
		"web": "false",
	}
	var redirectPaths []netv1.HTTPIngressPath
	pathType := netv1.PathTypePrefix

	hAProxyTimeout := pulp.Spec.HAProxyTimeout
	if len(hAProxyTimeout) == 0 {
		hAProxyTimeout = "180s"
	}
	redirectAnnotation["haproxy.router.openshift.io/timeout"] = hAProxyTimeout

	for _, plugin := range plugins {
		if len(plugin.Rewrite) > 0 {
			redirectAnnotation["haproxy.router.openshift.io/rewrite-target"] = plugin.Rewrite
			path := netv1.HTTPIngressPath{
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
			redirectPaths = append(redirectPaths, path)
		}
	}

	// if there is no plugin that needs a redirect, don't create the redirect route
	if len(redirectPaths) == 0 {
		return nil
	}

	for key, val := range pulp.Spec.IngressAnnotations {
		redirectAnnotation[key] = val
	}

	expectedIngress.ObjectMeta.Annotations = redirectAnnotation
	expectedIngress.ObjectMeta.Name = ingressName
	expectedIngress.Spec.IngressClassName = &pulp.Spec.IngressClassName
	expectedIngress.Spec.Rules[0].IngressRuleValue = netv1.IngressRuleValue{
		HTTP: &netv1.HTTPIngressRuleValue{
			Paths: redirectPaths,
		},
	}

	// [TODO] Refactor this. We should not be deploying the ingress here (through this function).
	// For now, keeping the same approach as of the old commits/implementation while I cannot find a
	// better way.
	currentIngress := &netv1.Ingress{}
	if err = resources.Get(resources.Context, types.NamespacedName{Name: ingressName, Namespace: pulp.Namespace}, currentIngress); err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new ingress", "Ingress.Namespace", expectedIngress.Namespace, "Ingress.Name", ingressName)
		controllers.UpdateStatus(resources.Context, resources.Client, pulp, metav1.ConditionFalse, conditionType, "CreatingIngress", "Creating "+pulp.Name+"-ingress")
		err = resources.Create(resources.Context, expectedIngress)
		if err != nil {
			log.Error(err, "Failed to create new ingress", "Ingress.Namespace", expectedIngress.Namespace, "Ingress.Name", ingressName)
			controllers.UpdateStatus(resources.Context, resources.Client, pulp, metav1.ConditionFalse, conditionType, "ErrorCreatingIngress", "Failed to create "+pulp.Name+"-ingress: "+err.Error())
			return err
		}
	} else if err != nil {
		log.Error(err, "Failed to get ingress")
		return err
	}

	// Ensure ingress specs are as expected
	if _, err := controllers.ReconcileObject(controllers.FunctionResources{Context: resources.Context, Client: resources.Client, Pulp: pulp, Scheme: resources.Scheme, Logger: log}, expectedIngress, currentIngress, conditionType, controllers.PulpIngress{}); err != nil {
		return err
	}

	// Ensure ingress labels and annotations are as expected
	if _, err := controllers.ReconcileMetadata(controllers.FunctionResources{Context: resources.Context, Client: resources.Client, Pulp: pulp, Scheme: resources.Scheme, Logger: log}, expectedIngress, currentIngress, conditionType); err != nil {
		return err
	}

	return nil
}

// deploy defines the default ocp ingress spec (rules to api and content services that don't need redirection)
func deployOCPIngress(resources controllers.FunctionResources, plugins []controllers.IngressPlugin) (*netv1.Ingress, error) {
	pulp := resources.Pulp
	var paths []netv1.HTTPIngressPath
	pathType := netv1.PathTypePrefix

	ingress, err := controllers.IngressDefaults(resources, plugins)
	if err != nil {
		return nil, err
	}

	for _, plugin := range plugins {
		if len(plugin.Rewrite) == 0 {
			path := netv1.HTTPIngressPath{
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
	}

	annotations := map[string]string{
		"web": "false",
	}
	hAProxyTimeout := pulp.Spec.HAProxyTimeout
	if len(hAProxyTimeout) == 0 {
		hAProxyTimeout = "180s"
	}
	annotations["haproxy.router.openshift.io/timeout"] = hAProxyTimeout

	for key, val := range pulp.Spec.IngressAnnotations {
		annotations[key] = val
	}

	ingress.ObjectMeta.Annotations = annotations
	ingress.Spec.IngressClassName = &pulp.Spec.IngressClassName
	ingress.Spec.Rules[0].HTTP.Paths = paths
	return ingress, nil
}

// UpdateIngressClass will handle the modifications needed when changing to/from "openshift-default" ingressclass
func UpdateIngressClass(resources controllers.FunctionResources) {

	pulp := resources.Pulp
	ctx := resources.Context

	// if the new IngressClass is an "openshift-default"
	if pulp.Spec.IngressClassName == controllers.DefaultOCPIngressClass {
		// remove pulp-web components
		controllers.RemovePulpWebResources(resources)
	}

	// if it was an Ingress with "openshift-default" IngressClass and is not anymore, remove "pulp-redirect" Ingress
	if pulp.Status.IngressClassName == controllers.DefaultOCPIngressClass && pulp.Spec.IngressClassName != controllers.DefaultOCPIngressClass {
		ingress := &netv1.Ingress{}
		if err := resources.Get(ctx, types.NamespacedName{Name: pulp.Name + "-redirect", Namespace: pulp.Namespace}, ingress); !errors.IsNotFound(err) {
			resources.Delete(ctx, ingress)
		}
	}
}

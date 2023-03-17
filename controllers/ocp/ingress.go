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
	netv1 "k8s.io/api/networking/v1"
)

// IngressOCP is the Ingress definition for OCP clusters
type IngressOCP struct{}

// Deploy returns an Ingress based on default OCP IngressController (openshift.io/ingress-to-route)
func (i IngressOCP) Deploy(resources controllers.FunctionResources, plugins []controllers.IngressPlugin) (*netv1.Ingress, error) {

	pulp := resources.Pulp

	ingress, err := controllers.IngressDefaults(resources, plugins)
	if err != nil {
		return nil, err
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

	for key, val := range pulp.Spec.IngressAnnotations {
		redirectAnnotation[key] = val
	}

	ingress.ObjectMeta.Annotations = redirectAnnotation
	ingress.Spec.IngressClassName = &pulp.Spec.IngressClassName
	ingress.Spec.Rules[0].IngressRuleValue = netv1.IngressRuleValue{
		HTTP: &netv1.HTTPIngressRuleValue{
			Paths: redirectPaths,
		},
	}

	return ingress, nil
}

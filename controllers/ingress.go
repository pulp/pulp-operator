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
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// IngressPlugin defines a plugin ingress.
type IngressPlugin struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	ServiceName string `json:"serviceName"`
	TargetPort  string `json:"targetPort"`
	Rewrite     string `json:"rewrite"`
}

// IngressDefaults returns an k8s Ingress resource with default values
func IngressDefaults(resources any, plugins []IngressPlugin) (*netv1.Ingress, error) {
	pulp := resources.(FunctionResources).Pulp
	annotation := map[string]string{
		"web": "false",
	}
	var paths []netv1.HTTPIngressPath
	var path netv1.HTTPIngressPath
	pathType := netv1.PathTypePrefix

	for _, plugin := range plugins {
		if len(plugin.Rewrite) > 0 {
			annotation["web"] = "true"
			path = netv1.HTTPIngressPath{
				Path:     "/",
				PathType: &pathType,
				Backend: netv1.IngressBackend{
					Service: &netv1.IngressServiceBackend{
						Name: pulp.Name + "-web-svc",
						Port: netv1.ServiceBackendPort{
							Number: 24880,
						},
					},
				},
			}
			paths = append(paths, path)
			break
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

	for key, val := range pulp.Spec.IngressAnnotations {
		annotation[key] = val
	}

	hostname := pulp.Spec.IngressHost
	if len(pulp.Spec.Hostname) > 0 { // [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
		hostname = pulp.Spec.Hostname
	}
	ingressSpec := netv1.IngressSpec{
		IngressClassName: &pulp.Spec.IngressClassName,
		Rules: []netv1.IngressRule{
			{
				Host: hostname,
				IngressRuleValue: netv1.IngressRuleValue{
					HTTP: &netv1.HTTPIngressRuleValue{
						Paths: paths,
					},
				},
			},
		},
	}

	if len(pulp.Spec.IngressTLSSecret) > 0 {
		ingressSpec.TLS = []netv1.IngressTLS{
			{
				Hosts:      []string{hostname},
				SecretName: pulp.Spec.IngressTLSSecret,
			},
		}
	}
	labels := map[string]string{
		"app.kubernetes.io/name":       "ingress",
		"app.kubernetes.io/instance":   "ingress-" + pulp.Name,
		"app.kubernetes.io/component":  "ingress",
		"app.kubernetes.io/part-of":    pulp.Spec.DeploymentType,
		"app.kubernetes.io/managed-by": pulp.Spec.DeploymentType + "-operator",
		"pulp_cr":                      pulp.Name,
		"owner":                        "pulp-dev",
	}

	ingress := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        pulp.Name,
			Namespace:   pulp.Namespace,
			Labels:      labels,
			Annotations: annotation,
		},
		Spec: ingressSpec,
	}
	ctrl.SetControllerReference(pulp, ingress, resources.(FunctionResources).Scheme)
	return ingress, nil
}

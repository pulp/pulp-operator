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
	"context"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	ctrl "sigs.k8s.io/controller-runtime"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
	"github.com/go-logr/logr"
)

func (r *PulpReconciler) pulpWebController(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, log logr.Logger) (ctrl.Result, error) {

	// pulp-web Configmap
	webConfigMap := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-configmap", Namespace: pulp.Namespace}, webConfigMap)
	newWebConfigMap := r.pulpWebConfigMap(pulp)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Pulp Web ConfigMap", "ConfigMap.Namespace", newWebConfigMap.Namespace, "ConfigMap.Name", newWebConfigMap.Name)
		r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-Web-Ready", "CreatingWebConfigmap", "Creating "+pulp.Name+"-web configmap resource")
		err = r.Create(ctx, newWebConfigMap)
		if err != nil {
			log.Error(err, "Failed to create new Pulp Web ConfigMap", "ConfigMap.Namespace", newWebConfigMap.Namespace, "ConfigMap.Name", newWebConfigMap.Name)
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-Web-Ready", "ErrorCreatingWebConfigmap", "Failed to create "+pulp.Name+"-web configmap resource: "+err.Error())
			return ctrl.Result{}, err
		}
		// ConfigMap created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Pulp Web ConfigMap")
		return ctrl.Result{}, err
	}

	// pulp-web Deployment
	webDeployment := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-web", Namespace: pulp.Namespace}, webDeployment)
	newWebDeployment := r.deploymentForPulpWeb(pulp)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Pulp Web Deployment", "Deployment.Namespace", newWebDeployment.Namespace, "Deployment.Name", newWebDeployment.Name)
		r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-Web-Ready", "CreatingWebDeployment", "Creating "+pulp.Name+"-web deployment resource")
		err = r.Create(ctx, newWebDeployment)
		if err != nil {
			log.Error(err, "Failed to create new Pulp Web Deployment", "Deployment.Namespace", newWebDeployment.Namespace, "Deployment.Name", newWebDeployment.Name)
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-Web-Ready", "ErrorCreatingWebDeployment", "Failed to create "+pulp.Name+"-web deployment resource: "+err.Error())
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Pulp Web Deployment")
		return ctrl.Result{}, err
	}

	// Reconcile Deployment
	if !equality.Semantic.DeepDerivative(newWebDeployment.Spec, webDeployment.Spec) {
		log.Info("The PULP-WEB Deployment has been modified! Reconciling ...")
		r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-Web-Ready", "UpdatingWebDeployment", "Reconciling "+pulp.Name+"-web deployment resource")
		err = r.Update(ctx, newWebDeployment)
		if err != nil {
			log.Error(err, "Error trying to update the PULP-WEB Deployment object ... ")
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-Web-Ready", "ErrorUpdatingWebDeployment", "Failed to reconcile "+pulp.Name+"-web deployment resource: "+err.Error())
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	// SERVICE
	webSvc := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-web-svc", Namespace: pulp.Namespace}, webSvc)
	newWebSvc := serviceForPulpWeb(pulp)
	if err != nil && errors.IsNotFound(err) {
		ctrl.SetControllerReference(pulp, newWebSvc, r.Scheme)
		log.Info("Creating a new Web Service", "Service.Namespace", newWebSvc.Namespace, "Service.Name", newWebSvc.Name)
		r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-Web-Ready", "CreatingWebDService", "Creating "+pulp.Name+"-web-svc service resource")
		err = r.Create(ctx, newWebSvc)
		if err != nil {
			log.Error(err, "Failed to create new Web Service", "Service.Namespace", newWebSvc.Namespace, "Service.Name", newWebSvc.Name)
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-Web-Ready", "ErrorCreatingWebDService", "Failed to create "+pulp.Name+"-web-svc service resource: "+err.Error())
			return ctrl.Result{}, err
		}
		// Service created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Web Service")
		return ctrl.Result{}, err
	}

	/* This reconcile is getting into an infinite loop.
	// Reconcile Service
	if !equality.Semantic.DeepDerivative(newWebSvc.Spec, webSvc.Spec) {
		log.Info("The PULP-WEB Service has been modified! Reconciling ...")
		ctrl.SetControllerReference(pulp, newWebSvc, r.Scheme)
		err = r.Update(ctx, newWebSvc)
		if err != nil {
			log.Error(err, "Error trying to update the PULP-WEB Service object ... ")
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}
	*/

	r.updateStatus(ctx, pulp, metav1.ConditionTrue, pulp.Spec.DeploymentType+"-Web-Ready", "WebTasksFinished", "All Web tasks ran successfully")
	return ctrl.Result{}, nil
}

// deploymentForPulpWeb returns a pulp-web Deployment object
func (r *PulpReconciler) deploymentForPulpWeb(m *repomanagerv1alpha1.Pulp) *appsv1.Deployment {

	runAsUser := int64(0)
	fsGroup := int64(0)

	ls := labelsForPulpWeb(m)
	replicas := m.Spec.Web.Replicas
	resources := m.Spec.Web.ResourceRequirements

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-web",
			Namespace: m.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "nginx",
				"app.kubernetes.io/instance":   "nginx-" + m.Name,
				"app.kubernetes.io/component":  "webserver",
				"app.kubernetes.io/part-of":    m.Spec.DeploymentType,
				"app.kubernetes.io/managed-by": m.Spec.DeploymentType + "-operator",
				"owner":                        "pulp-dev",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:     m.Spec.ImageWeb + ":" + m.Spec.ImageWebVersion,
						Name:      "web",
						Resources: resources,
						Env: []corev1.EnvVar{
							{
								Name: "NODE_IP",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "status.hostIP",
									},
								},
							},
						},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 8080,
							Protocol:      "TCP",
						}},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      m.Name + "-nginx-conf",
								MountPath: "/etc/nginx/nginx.conf",
								SubPath:   "nginx.conf",
								ReadOnly:  true,
							},
						},
					}},
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser: &runAsUser,
						FSGroup:   &fsGroup,
					},
					Volumes: []corev1.Volume{
						{
							Name: m.Name + "-nginx-conf",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: m.Name + "-configmap",
									},
									Items: []corev1.KeyToPath{
										{Key: "nginx.conf", Path: "nginx.conf"},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

// labelsForPulpWeb returns the labels for selecting the resources
// belonging to the given pulp CR name.
func labelsForPulpWeb(m *repomanagerv1alpha1.Pulp) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "nginx",
		"app.kubernetes.io/instance":   "nginx-" + m.Name,
		"app.kubernetes.io/component":  "webserver",
		"app.kubernetes.io/part-of":    m.Spec.DeploymentType,
		"app.kubernetes.io/managed-by": m.Spec.DeploymentType + "-operator",
		"pulp_cr":                      m.Name,
	}
}

// serviceForPulpWeb returns a service object for pulp-web
func serviceForPulpWeb(m *repomanagerv1alpha1.Pulp) *corev1.Service {
	var serviceType corev1.ServiceType
	servicePort := []corev1.ServicePort{{
		Port:       24880,
		Protocol:   corev1.ProtocolTCP,
		TargetPort: intstr.IntOrString{IntVal: 8080},
		Name:       "web-8080",
	}}

	if strings.ToLower(m.Spec.IngressType) == "nodeport" {
		serviceType = corev1.ServiceType(corev1.ServiceTypeNodePort)
		if m.Spec.NodePort > 0 {
			servicePort[0].NodePort = m.Spec.NodePort
		}
	} else {
		serviceType = corev1.ServiceType(corev1.ServiceTypeClusterIP)
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-web-svc",
			Namespace: m.Namespace,
			Labels:    labelsForPulpWeb(m),
		},
		Spec: corev1.ServiceSpec{
			Selector: labelsForPulpWeb(m),
			Ports:    servicePort,
			Type:     serviceType,
		},
	}
}

// wouldn't it be better to handle the configmap content by loading it from a file?
func (r *PulpReconciler) pulpWebConfigMap(m *repomanagerv1alpha1.Pulp) *corev1.ConfigMap {
	sec := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-configmap",
			Namespace: m.Namespace,
		},
		Data: map[string]string{
			"nginx.conf": `
error_log /dev/stdout info;
worker_processes 1;
events {
	worker_connections 1024;  # increase if you have lots of clients
	accept_mutex off;  # set to 'on' if nginx worker_processes > 1
}

http {
	access_log /dev/stdout;
	include mime.types;
	# fallback in case we can't determine a type
	default_type application/octet-stream;
	sendfile on;

	# If left at the default of 1024, nginx emits a warning about being unable
	# to build optimal hash types.
	types_hash_max_size 4096;

	upstream pulp-content {
		server ` + m.Name + `-content-svc:24816;
	}

	upstream pulp-api {
		server ` + m.Name + `-api-svc:24817;
	}

	server {

		# Gunicorn docs suggest the use of the "deferred" directive on Linux.
		listen 8080 default_server deferred;
		listen [::]:8080 default_server deferred;

		# If you have a domain name, this is where to add it
		server_name $hostname;

		proxy_read_timeout 120s;
		proxy_connect_timeout 120s;
		proxy_send_timeout 120s;

		# The default client_max_body_size is 1m. Clients uploading
		# files larger than this will need to chunk said files.
		client_max_body_size 10m;

		# Gunicorn docs suggest this value.
		keepalive_timeout 5;

		# static files that can change dynamically, or are needed for TLS
		# purposes are served through the webserver.
		root "/opt/app-root/src";

		location /pulp/content/ {
			proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
			proxy_set_header X-Forwarded-Proto $scheme;
			proxy_set_header Host $http_host;
			# we don't want nginx trying to do something clever with
			# redirects, we set the Host: header above already.
			proxy_redirect off;
			proxy_pass http://pulp-content;
		}

		location ` + m.Spec.PulpSettings.ApiRoot + `api/v3/ {
			proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
			proxy_set_header X-Forwarded-Proto $scheme;
			proxy_set_header Host $http_host;
			# we don't want nginx trying to do something clever with
			# redirects, we set the Host: header above already.
			proxy_redirect off;
			proxy_pass http://pulp-api;
		}

		location /auth/login/ {
			proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
			proxy_set_header X-Forwarded-Proto $scheme;
			proxy_set_header Host $http_host;
			# we don't want nginx trying to do something clever with
			# redirects, we set the Host: header above already.
			proxy_redirect off;
			proxy_pass http://pulp-api;
		}

		include /opt/app-root/etc/nginx.default.d/*.conf;

		location / {
			proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
			proxy_set_header X-Forwarded-Proto $scheme;
			proxy_set_header Host $http_host;
			# we don't want nginx trying to do something clever with
			# redirects, we set the Host: header above already.
			proxy_redirect off;
			proxy_pass http://pulp-api;
			# static files are served through whitenoise - http://whitenoise.evans.io/en/stable/
		}
	}
}
			`,
		},
	}

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(m, sec, r.Scheme)
	return sec
}

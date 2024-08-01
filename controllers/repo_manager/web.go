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
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"
	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	"github.com/pulp/pulp-operator/controllers"
	"github.com/pulp/pulp-operator/controllers/settings"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *RepoManagerReconciler) pulpWebController(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp, log logr.Logger) (ctrl.Result, error) {
	funcResources := controllers.FunctionResources{Context: ctx, Client: r.Client, Pulp: pulp, Scheme: r.Scheme, Logger: log}

	// conditionType is used to update .status.conditions with the current resource state
	conditionType := cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Web-Ready"

	// pulp-web Configmap
	configMapName := settings.PulpWebConfigMapName(pulp.Name)
	webConfigMap := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: pulp.Namespace}, webConfigMap)
	newWebConfigMap := r.pulpWebConfigMap(pulp)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Pulp Web ConfigMap", "ConfigMap.Namespace", newWebConfigMap.Namespace, "ConfigMap.Name", newWebConfigMap.Name)
		controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, conditionType, "CreatingWebConfigmap", "Creating "+pulp.Name+"-web configmap resource")
		err = r.Create(ctx, newWebConfigMap)
		if err != nil {
			log.Error(err, "Failed to create new Pulp Web ConfigMap", "ConfigMap.Namespace", newWebConfigMap.Namespace, "ConfigMap.Name", newWebConfigMap.Name)
			controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, conditionType, "ErrorCreatingWebConfigmap", "Failed to create "+pulp.Name+"-web configmap resource: "+err.Error())
			r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to create new Web ConfigMap")
			return ctrl.Result{}, err
		}
		// ConfigMap created successfully - return and requeue
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Created", "Web ConfigMap created")
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Pulp Web ConfigMap")
		return ctrl.Result{}, err
	}

	// pulp-web Deployment
	deploymentName := settings.WEB.DeploymentName(pulp.Name)
	webDeployment := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: pulp.Namespace}, webDeployment)
	newWebDeployment := r.deploymentForPulpWeb(pulp, funcResources)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Pulp Web Deployment", "Deployment.Namespace", newWebDeployment.Namespace, "Deployment.Name", newWebDeployment.Name)
		controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, conditionType, "CreatingWebDeployment", "Creating "+deploymentName+" Deployment resource")
		err = r.Create(ctx, newWebDeployment)
		if err != nil {
			log.Error(err, "Failed to create new Pulp Web Deployment", "Deployment.Namespace", newWebDeployment.Namespace, "Deployment.Name", newWebDeployment.Name)
			controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, conditionType, "ErrorCreatingWebDeployment", "Failed to create "+deploymentName+" Deployment resource: "+err.Error())
			r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to create new Web Deployment")
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Created", "Web Deployment created")
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Pulp Web Deployment")
		return ctrl.Result{}, err
	}

	// Reconcile Deployment
	if controllers.CheckDeploymentSpec(*newWebDeployment, *webDeployment, funcResources) {
		log.Info("The Web Deployment has been modified! Reconciling ...")
		controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, conditionType, "UpdatingWebDeployment", "Reconciling "+deploymentName+" Deployment resource")
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Updating", "Reconciling Web Deployment")
		err = r.Update(ctx, newWebDeployment)
		if err != nil {
			log.Error(err, "Error trying to update the Web Deployment object ... ")
			controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, conditionType, "ErrorUpdatingWebDeployment", "Failed to reconcile "+deploymentName+" Deployment resource: "+err.Error())
			r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to reconcile Web Deployment")
			return ctrl.Result{}, err
		}
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Updated", "Web Deployment reconciled")
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second}, nil
	}

	// SERVICE
	serviceName := settings.PulpWebService(pulp.Name)
	webSvc := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: pulp.Namespace}, webSvc)
	newWebSvc := serviceForPulpWeb(pulp)
	if err != nil && errors.IsNotFound(err) {
		ctrl.SetControllerReference(pulp, newWebSvc, r.Scheme)
		log.Info("Creating a new Web Service", "Service.Namespace", newWebSvc.Namespace, "Service.Name", newWebSvc.Name)
		controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, conditionType, "CreatingWebService", "Creating "+serviceName+" Service resource")
		err = r.Create(ctx, newWebSvc)
		if err != nil {
			log.Error(err, "Failed to create new Web Service", "Service.Namespace", newWebSvc.Namespace, "Service.Name", newWebSvc.Name)
			controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, conditionType, "ErrorCreatingWebDService", "Failed to create "+serviceName+" Service resource: "+err.Error())
			r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to create new Web Service")
			return ctrl.Result{}, err
		}
		// Service created successfully - return and requeue
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Created", "Web Service created")
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Web Service")
		return ctrl.Result{}, err
	}

	// Reconcile Service
	if requeue, err := controllers.ReconcileObject(controllers.FunctionResources{Context: ctx, Client: r.Client, Pulp: pulp, Scheme: r.Scheme, Logger: log}, newWebSvc, webSvc, conditionType, controllers.PulpService{}); err != nil || requeue {
		return ctrl.Result{Requeue: requeue}, err
	}

	return ctrl.Result{}, nil
}

// deploymentForPulpWeb returns a pulp-web Deployment object
func (r *RepoManagerReconciler) deploymentForPulpWeb(m *repomanagerpulpprojectorgv1beta2.Pulp, funcResources controllers.FunctionResources) *appsv1.Deployment {

	ls := labelsForPulpWeb(m)
	replicas := m.Spec.Web.Replicas
	resources := m.Spec.Web.ResourceRequirements
	ImageWeb := os.Getenv("RELATED_IMAGE_PULP_WEB")
	if len(m.Spec.ImageWeb) > 0 && len(m.Spec.ImageWebVersion) > 0 {
		ImageWeb = m.Spec.ImageWeb + ":" + m.Spec.ImageWebVersion
	} else if ImageWeb == "" {
		ImageWeb = "quay.io/pulp/pulp-web:stable"
	}

	// if no strategy is defined in pulp CR we are setting `strategy.Type` with the
	// default value ("RollingUpdate"), this will be helpful during the reconciliation
	// when a strategy was previously defined and eventually the field is removed
	strategy := m.Spec.Web.Strategy
	if strategy.Type == "" {
		strategy.Type = "RollingUpdate"
	}

	readinessProbe := m.Spec.Web.ReadinessProbe
	if readinessProbe == nil {
		readinessProbe = &corev1.Probe{
			FailureThreshold: 2,
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: controllers.GetAPIRoot(funcResources.Client, m) + "api/v3/status/",
					Port: intstr.IntOrString{
						IntVal: 8080,
					},
					Scheme: corev1.URIScheme("HTTP"),
				},
			},
			InitialDelaySeconds: 3,
			PeriodSeconds:       10,
			SuccessThreshold:    1,
			TimeoutSeconds:      10,
		}
	}

	livenessProbe := m.Spec.Web.LivenessProbe

	nodeSelector := map[string]string{}
	if m.Spec.Web.NodeSelector != nil {
		nodeSelector = m.Spec.Web.NodeSelector
	}

	envVars := []corev1.EnvVar{
		{
			Name: "NODE_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.hostIP",
				},
			},
		},
	}
	envVars = append(envVars, m.Spec.Web.EnvVars...)

	runAsUser := int64(700)
	fsGroup := int64(700)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        settings.WEB.DeploymentName(m.Name),
			Namespace:   m.Namespace,
			Labels:      ls,
			Annotations: m.Spec.Web.DeploymentAnnotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Strategy: strategy,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					NodeSelector:       nodeSelector,
					ServiceAccountName: settings.PulpServiceAccount(m.Name),
					Containers: []corev1.Container{{
						Image:     ImageWeb,
						Name:      "web",
						Resources: resources,
						Env:       envVars,
						Ports: []corev1.ContainerPort{{
							ContainerPort: 8080,
							Protocol:      "TCP",
						}},
						LivenessProbe:  livenessProbe,
						ReadinessProbe: readinessProbe,
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      m.Name + "-nginx-conf",
								MountPath: "/etc/nginx/nginx.conf",
								SubPath:   "nginx.conf",
								ReadOnly:  true,
							},
						},
						SecurityContext: controllers.SetDefaultSecurityContext(),
					}},
					SecurityContext: &corev1.PodSecurityContext{RunAsUser: &runAsUser, FSGroup: &fsGroup},
					Volumes: []corev1.Volume{
						{
							Name: m.Name + "-nginx-conf",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: settings.PulpWebConfigMapName(m.Name),
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

	controllers.AddHashLabel(funcResources, dep)
	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

// labelsForPulpWeb returns the labels for selecting the resources
// belonging to the given pulp CR name.
func labelsForPulpWeb(m *repomanagerpulpprojectorgv1beta2.Pulp) map[string]string {
	return settings.PulpcoreLabels(*m, "web")
}

// serviceForPulpWeb returns a service object for pulp-web
func serviceForPulpWeb(m *repomanagerpulpprojectorgv1beta2.Pulp) *corev1.Service {
	annotations := m.Spec.Web.ServiceAnnotations

	var serviceType corev1.ServiceType
	servicePort := []corev1.ServicePort{}

	lbPort := int32(80)
	lbProtocol := "http"
	if len(m.Spec.LoadbalancerProtocol) > 0 {
		lbProtocol = strings.ToLower(m.Spec.LoadbalancerProtocol)
	}

	ingressType := strings.ToLower(m.Spec.IngressType)
	if ingressType != "loadbalancer" && lbProtocol != "https" {
		port := corev1.ServicePort{
			Port:       24880,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.IntOrString{IntVal: 8080},
			Name:       "web-8080",
		}
		servicePort = append(servicePort, port)
	}
	if ingressType == "loadbalancer" && lbProtocol == "https" {
		lbPort = int32(443)
		if m.Spec.LoadbalancerPort != 0 {
			lbPort = m.Spec.LoadbalancerPort
		}
		port := corev1.ServicePort{
			Port:       lbPort,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.IntOrString{IntVal: 8080},
			Name:       "web-8443",
		}
		servicePort = append(servicePort, port)
	} else if ingressType == "loadbalancer" && lbProtocol != "https" {
		if m.Spec.LoadbalancerPort != 0 {
			lbPort = m.Spec.LoadbalancerPort
		}
		port := corev1.ServicePort{
			Port:       lbPort,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.IntOrString{IntVal: 8080},
			Name:       "web-8080",
		}
		servicePort = append(servicePort, port)
	}

	if strings.ToLower(m.Spec.IngressType) == "loadbalancer" {
		serviceType = corev1.ServiceType(corev1.ServiceTypeLoadBalancer)
	} else if strings.ToLower(m.Spec.IngressType) == "nodeport" {
		serviceType = corev1.ServiceType(corev1.ServiceTypeNodePort)
		if m.Spec.NodePort > 0 {
			servicePort[0].NodePort = m.Spec.NodePort
		}
	} else {
		serviceType = corev1.ServiceType(corev1.ServiceTypeClusterIP)
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        settings.PulpWebService(m.Name),
			Namespace:   m.Namespace,
			Labels:      labelsForPulpWeb(m),
			Annotations: annotations,
		},
		Spec: corev1.ServiceSpec{
			Selector: labelsForPulpWeb(m),
			Ports:    servicePort,
			Type:     serviceType,
		},
	}
}

// wouldn't it be better to handle the configmap content by loading it from a file?
func (r *RepoManagerReconciler) pulpWebConfigMap(m *repomanagerpulpprojectorgv1beta2.Pulp) *corev1.ConfigMap {

	// Nginx default values
	nginxProxyReadTimeout := m.Spec.NginxProxyReadTimeout
	if len(nginxProxyReadTimeout) == 0 {
		nginxProxyReadTimeout = "120s"
	}
	nginxProxyConnectTimeout := m.Spec.NginxProxyConnectTimeout
	if len(nginxProxyConnectTimeout) == 0 {
		nginxProxyConnectTimeout = "120s"
	}
	nginxProxySendTimeout := m.Spec.NginxProxySendTimeout
	if len(nginxProxySendTimeout) == 0 {
		nginxProxySendTimeout = "120s"
	}
	nginxMaxBodySize := m.Spec.NginxMaxBodySize
	if len(nginxMaxBodySize) == 0 {
		nginxMaxBodySize = "10m"
	}

	serverConfig := ""
	tlsTerminationMechanism := "edge"
	if len(m.Spec.Web.TLSTerminationMechanism) > 0 {
		tlsTerminationMechanism = strings.ToLower(m.Spec.Web.TLSTerminationMechanism)
	}

	if tlsTerminationMechanism == "passthrough" {
		serverConfig = `

    server {
        listen 8080 default_server;
        listen [::]:8080 default_server;
        server_name _;

        proxy_read_timeout ` + nginxProxyReadTimeout + `;
        proxy_connect_timeout ` + nginxProxyConnectTimeout + `;
        proxy_send_timeout ` + nginxProxySendTimeout + `;

        client_max_body_size ` + nginxMaxBodySize + `;

        # Redirect all HTTP links to the matching HTTPS page
        return 301 https://$host$request_uri;
    }

    server {
        listen 8443 default_server deferred ssl;
        listen [::]:8443 default_server deferred ssl;

        ssl_certificate /etc/nginx/pki/web.crt;
        ssl_certificate_key /etc/nginx/pki/web.key;
        ssl_session_cache shared:SSL:50m;
        ssl_session_timeout 1d;
        ssl_session_tickets off;

        # intermediate configuration
        ssl_protocols TLSv1.2;
        ssl_ciphers'ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:ECDHE-ECDS-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-SHA384:ECDHE-RSA-AES256-SHA384:ECDHE-ECDSA-AES128-SHA256:ECDH-RSA-AES128-SHA256';
        ssl_prefer_server_ciphers on;
`
	} else {
		serverConfig = `

    server {
    	# Gunicorn docs suggest the use of the "deferred" directive on Linux.
    	listen 8080 default_server deferred;
    	listen [::]:8080 default_server deferred;
`
	}

	data := map[string]string{
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
			server ` + settings.ContentService(m.Name) + `:24816;
		}

		upstream pulp-api {
			server ` + settings.ApiService(m.Name) + `:24817;
		}
` + serverConfig + `

			# If you have a domain name, this is where to add it
			server_name $hostname;

			proxy_read_timeout ` + nginxProxyReadTimeout + `;
			proxy_connect_timeout ` + nginxProxyConnectTimeout + `;
			proxy_send_timeout ` + nginxProxySendTimeout + `;

			# The default client_max_body_size is 1m. Clients uploading
			# files larger than this will need to chunk said files.
			client_max_body_size ` + nginxMaxBodySize + `;

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

			location ` + controllers.GetAPIRoot(r.Client, m) + `api/v3/ {
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
	}

	sec := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      settings.PulpWebConfigMapName(m.Name),
			Namespace: m.Namespace,
			Labels:    settings.CommonLabels(*m),
		},
		Data: data,
	}

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(m, sec, r.Scheme)
	return sec
}

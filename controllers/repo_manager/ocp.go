package repo_manager

import (
	"context"

	"github.com/go-logr/logr"
	configv1 "github.com/openshift/api/config/v1"
	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// createRHOperatorPullSecret creates a default secret called redhat-operators-pull-secret
func (r *RepoManagerReconciler) createRHOperatorPullSecret(ctx context.Context, namespace string) error {
	log := r.RawLogger

	// Get redhat-operators-pull-secret
	defaultSecret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: "redhat-operators-pull-secret", Namespace: namespace}, defaultSecret)

	// Create the secret in case it is not found
	if err != nil && k8s_errors.IsNotFound(err) {
		defaultSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "redhat-operators-pull-secret",
				Namespace: namespace,
			},
			StringData: map[string]string{
				"operator": "pulp",
			},
		}
		r.Create(ctx, defaultSecret)
	} else if err != nil {
		log.Error(err, "Failed to get redhat-operators-pull-secret")
		return err
	}
	return nil
}

// createEmptyConfigMap creates an empty ConfigMap that is used by CNO (Cluster Network Operator) to
// inject custom CA into containers
func (r *RepoManagerReconciler) createEmptyConfigMap(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp, log logr.Logger) (ctrl.Result, error) {

	configMap := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: caConfigMapName, Namespace: pulp.Namespace}, configMap)

	expected_cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      caConfigMapName,
			Namespace: pulp.Namespace,
			Labels: map[string]string{
				"config.openshift.io/inject-trusted-cabundle": "true",
			},
		},
		Data: map[string]string{},
	}

	// create the configmap if not found
	if err != nil && k8s_errors.IsNotFound(err) {
		log.V(1).Info("Creating a new empty ConfigMap")
		ctrl.SetControllerReference(pulp, expected_cm, r.Scheme)
		err = r.Create(ctx, expected_cm)
		if err != nil {
			log.Error(err, "Failed to create empty ConfigMap")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get empty ConfigMap")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// mountCASpec adds the trusted-ca bundle into []volume and []volumeMount if pulp.Spec.TrustedCA is true
func mountCASpec(pulp *repomanagerpulpprojectorgv1beta2.Pulp, volumes []corev1.Volume, volumeMounts []corev1.VolumeMount) ([]corev1.Volume, []corev1.VolumeMount) {

	if pulp.Spec.TrustedCa {

		// trustedCAVolume contains the configmap with the custom ca bundle
		trustedCAVolume := corev1.Volume{
			Name: "trusted-ca",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: caConfigMapName,
					},
					Items: []corev1.KeyToPath{
						{Key: "ca-bundle.crt", Path: "tls-ca-bundle.pem"},
					},
				},
			},
		}
		volumes = append(volumes, trustedCAVolume)

		// trustedCAMount defines the mount point of the configmap
		// with the custom ca bundle
		trustedCAMount := corev1.VolumeMount{
			Name:      "trusted-ca",
			MountPath: "/etc/pki/ca-trust/extracted/pem",
			ReadOnly:  true,
		}
		volumeMounts = append(volumeMounts, trustedCAMount)
	}

	return volumes, volumeMounts
}

// define route host based on ingress default cluster domain if no .spec.route_host defined
func getRouteHost(resource FunctionResources) string {
	routeHost := resource.Pulp.Spec.RouteHost
	if len(resource.Pulp.Spec.RouteHost) == 0 {
		ingress := &configv1.Ingress{}
		resource.RepoManagerReconciler.Get(resource.Context, types.NamespacedName{Name: "cluster"}, ingress)
		routeHost = resource.Pulp.Name + "." + ingress.Spec.Domain
	}
	return routeHost
}

// defaultsForOCPDeployment sets the common deployment configurations specific to OCP clusters
func defaultsForOCPDeployment(deployment *appsv1.Deployment, resources FunctionResources) {
	// in OCP we use SCC so there is no need to define PodSecurityContext
	deployment.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{}

	// get the current volume mount points
	volumes := deployment.Spec.Template.Spec.Volumes
	volumeMounts := deployment.Spec.Template.Spec.Containers[0].VolumeMounts

	// append the CA configmap to the volumes/volumemounts slice
	volumes, volumeMounts = mountCASpec(resources.Pulp, volumes, volumeMounts)
	deployment.Spec.Template.Spec.Volumes = volumes
	deployment.Spec.Template.Spec.Containers[0].VolumeMounts = volumeMounts
}

type DeploymentAPIOCP struct {
	DeploymentAPIVanilla
}

// deploy will set the specific OCP configurations for Pulp API deployment
func (d DeploymentAPIOCP) deploy(resources FunctionResources) client.Object {

	// get the current vanilla deployment definition
	deployment := d.DeploymentAPIVanilla.deploy(resources).(*appsv1.Deployment)
	defaultsForOCPDeployment(deployment, resources)
	return deployment
}

type DeploymentContentOCP struct {
	DeploymentContentVanilla
}

// deploy will set the specific OCP configurations for Pulp content deployment
func (d DeploymentContentOCP) deploy(resources FunctionResources) client.Object {

	// get the current vanilla deployment definition
	deployment := d.DeploymentContentVanilla.deploy(resources).(*appsv1.Deployment)
	defaultsForOCPDeployment(deployment, resources)
	return deployment
}

type DeploymentWorkerOCP struct {
	DeploymentWorkerVanilla
}

// deploy will set the specific OCP configurations for Pulp worker deployment
func (d DeploymentWorkerOCP) deploy(resources FunctionResources) client.Object {

	// get the current vanilla deployment definition
	deployment := d.DeploymentWorkerVanilla.deploy(resources).(*appsv1.Deployment)
	defaultsForOCPDeployment(deployment, resources)
	return deployment
}

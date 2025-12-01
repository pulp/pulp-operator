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
	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	"github.com/pulp/pulp-operator/controllers"
	pulp_ocp "github.com/pulp/pulp-operator/controllers/ocp"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type deploymentType int

const (
	API_DEPLOYMENT deploymentType = iota
	CONTENT_DEPLOYMENT
	WORKER_DEPLOYMENT
)

// DeploymentObj represents the k8s "Deployment" resource
type DeploymentObj struct {
	// Deployer is the abstraction for the different pulp deployment types (api,content,worker)
	Deployer
}

// initDeployment returns a Deployment object of type "deployer" based on k8s distribution and
// Pulp Deployment type (api,worker or content)
func initDeployment(dt deploymentType) *DeploymentObj {

	isOpenshift, _ := controllers.IsOpenShift()

	switch dt {
	case API_DEPLOYMENT:
		if isOpenshift {
			return &DeploymentObj{pulp_ocp.DeploymentAPIOCP{}}
		}
		return &DeploymentObj{DeploymentAPIVanilla{}}
	case WORKER_DEPLOYMENT:
		if isOpenshift {
			return &DeploymentObj{pulp_ocp.DeploymentWorkerOCP{}}
		}
		return &DeploymentObj{DeploymentWorkerVanilla{}}
	case CONTENT_DEPLOYMENT:
		if isOpenshift {
			return &DeploymentObj{pulp_ocp.DeploymentContentOCP{}}
		}
		return &DeploymentObj{DeploymentContentVanilla{}}
	}

	return &DeploymentObj{}
}

// Deployer is an interface for the several deployment types:
// - api Deployment in vanilla k8s or OCP
// - content Deployment in vanilla k8s or OCP
// - worker Deployment in vanilla k8s or OCP
type Deployer interface {
	Deploy(controllers.FunctionResources) client.Object
}

// defaultsForVanillaDeployment sets the common Deployment configurations for vanilla k8s clusters
// This includes CA bundle mounting from ConfigMaps when configured
func defaultsForVanillaDeployment(deployment client.Object, pulp *pulpv1.Pulp) {
	dep := deployment.(*appsv1.Deployment)

	// Validate CA ConfigMap configuration on vanilla K8s
	if pulp.Spec.TrustedCa && len(pulp.Spec.TrustedCaConfigMapKey) == 0 {
		controllers.CustomZapLogger().Error(`mount_trusted_ca is true but mount_trusted_ca_configmap_key is not set. ` +
			`On vanilla Kubernetes, you must specify mount_trusted_ca_configmap_key to reference a ConfigMap containing CA certificates. ` +
			`This field is only optional on OpenShift where CNO injection is used.`)
		return
	}

	// Only mount CA bundles if ConfigMap is specified
	if pulp.Spec.TrustedCa && len(pulp.Spec.TrustedCaConfigMapKey) > 0 {
		// get the current volume mount points
		volumes := dep.Spec.Template.Spec.Volumes
		volumeMounts := dep.Spec.Template.Spec.Containers[0].VolumeMounts

		// append the CA configmap to the volumes/volumemounts slice
		volumes, volumeMounts = pulp_ocp.MountCASpec(pulp, volumes, volumeMounts)
		dep.Spec.Template.Spec.Volumes = volumes
		dep.Spec.Template.Spec.Containers[0].VolumeMounts = volumeMounts
	}
}

// DeploymentAPIVanilla is the pulpcore-api Deployment definition for common k8s distributions
type DeploymentAPIVanilla struct{}

// Deploy returns a pulp-api Deployment object
func (DeploymentAPIVanilla) Deploy(resources controllers.FunctionResources) client.Object {
	dep := controllers.DeploymentAPICommon{}
	deployment := dep.Deploy(resources)
	defaultsForVanillaDeployment(deployment, resources.Pulp)
	return deployment
}

// DeploymentContentVanilla is the pulpcore-content Deployment definition for common k8s distributions
type DeploymentContentVanilla struct{}

// Deploy returns a pulp-content Deployment object
func (DeploymentContentVanilla) Deploy(resources controllers.FunctionResources) client.Object {
	dep := controllers.DeploymentContentCommon{}
	deployment := dep.Deploy(resources)
	defaultsForVanillaDeployment(deployment, resources.Pulp)
	return deployment
}

// DeploymentWorkerVanilla is the pulpcore-worker Deployment definition for common k8s distributions
type DeploymentWorkerVanilla struct{}

// Deploy returns a pulp-worker Deployment object
func (DeploymentWorkerVanilla) Deploy(resources controllers.FunctionResources) client.Object {
	dep := controllers.DeploymentWorkerCommon{}
	deployment := dep.Deploy(resources)
	defaultsForVanillaDeployment(deployment, resources.Pulp)
	return deployment
}

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
	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	"github.com/pulp/pulp-operator/controllers"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// defaultsForOCPDeployment sets the common Deployment configurations specific to OCP clusters
func defaultsForOCPDeployment(deployment *appsv1.Deployment, pulp *pulpv1.Pulp) {
	// in OCP we use SCC so there is no need to define PodSecurityContext
	deployment.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{}
}

// DeploymentAPIOCP is the pulpcore-api Deployment definition for common OCP clusters
type DeploymentAPIOCP struct {
	controllers.DeploymentAPICommon
}

// Deploy will set the specific OCP configurations for Pulp API Deployment
func (d DeploymentAPIOCP) Deploy(resources controllers.FunctionResources) client.Object {

	// get the current pulpcore-api common deployment definition
	deployment := d.DeploymentAPICommon.Deploy(resources).(*appsv1.Deployment)
	defaultsForOCPDeployment(deployment, resources.Pulp)

	// update the hash label
	controllers.AddHashLabel(resources, deployment)

	return deployment
}

// DeploymentContentOCP is the pulpcore-content Deployment definition for common OCP clusters
type DeploymentContentOCP struct {
	controllers.DeploymentContentCommon
}

// Deploy will set the specific OCP configurations for Pulp content Deployment
func (d DeploymentContentOCP) Deploy(resources controllers.FunctionResources) client.Object {

	// get the current pulpcore-content common deployment definition
	deployment := d.DeploymentContentCommon.Deploy(resources).(*appsv1.Deployment)
	defaultsForOCPDeployment(deployment, resources.Pulp)

	// update the hash label
	controllers.AddHashLabel(resources, deployment)

	return deployment
}

// DeploymentWorkerOCP is the pulpcore-worker Deployment definition for common OCP clusters
type DeploymentWorkerOCP struct {
	controllers.DeploymentWorkerCommon
}

// Deploy will set the specific OCP configurations for Pulp worker Deployment
func (d DeploymentWorkerOCP) Deploy(resources controllers.FunctionResources) client.Object {

	// get the current pulpcore-worker common deployment definition
	deployment := d.DeploymentWorkerCommon.Deploy(resources).(*appsv1.Deployment)
	defaultsForOCPDeployment(deployment, resources.Pulp)

	// update the hash label
	controllers.AddHashLabel(resources, deployment)

	return deployment
}

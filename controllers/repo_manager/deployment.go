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
	"github.com/pulp/pulp-operator/controllers"
	pulp_ocp "github.com/pulp/pulp-operator/controllers/ocp"
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

// DeploymentAPIVanilla is the pulpcore-api Deployment definition for common k8s distributions
type DeploymentAPIVanilla struct{}

// Deploy returns a pulp-api Deployment object
func (DeploymentAPIVanilla) Deploy(resources controllers.FunctionResources) client.Object {
	dep := controllers.DeploymentAPICommon{}
	return dep.Deploy(resources)
}

// DeploymentContentVanilla is the pulpcore-content Deployment definition for common k8s distributions
type DeploymentContentVanilla struct{}

// Deploy returns a pulp-content Deployment object
func (DeploymentContentVanilla) Deploy(resources controllers.FunctionResources) client.Object {
	dep := controllers.DeploymentContentCommon{}
	return dep.Deploy(resources)
}

// DeploymentWorkerVanilla is the pulpcore-worker Deployment definition for common k8s distributions
type DeploymentWorkerVanilla struct{}

// Deploy returns a pulp-worker Deployment object
func (DeploymentWorkerVanilla) Deploy(resources controllers.FunctionResources) client.Object {
	dep := controllers.DeploymentWorkerCommon{}
	return dep.Deploy(resources)
}

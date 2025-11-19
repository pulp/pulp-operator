// This file contains resource names and constants that are used to provision
// the Kubernetes objects. We are centralizing them here to make it easier to
// maintain and, in case we decide to support multiple CRs running in the same
// namespace, to avoid name colision or code repetition.
// Since go const does not allow to pass variables and there is no immutable vars
// we are encapsulating the constants in each function to return a value based
// on Pulp CR name.

package settings

import "strings"

type PulpcoreType string

const (
	API      PulpcoreType = "Api"
	CONTENT  PulpcoreType = "Content"
	WORKER   PulpcoreType = "Worker"
	WEB      PulpcoreType = "Web"
	CACHE    PulpcoreType = "Redis"
	DATABASE PulpcoreType = "Database"
)

func (t PulpcoreType) DeploymentName(pulpName string) string {
	return pulpName + "-" + strings.ToLower(string(t))
}

func (t PulpcoreType) ToField() string {
	if t == CACHE {
		return "Cache"
	}

	return string(t)
}

func (t PulpcoreType) ToLabel() string {
	if t == CACHE {
		return "cache"
	}

	return strings.ToLower(string(t))
}

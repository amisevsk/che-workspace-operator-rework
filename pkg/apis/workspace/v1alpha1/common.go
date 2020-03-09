package v1alpha1

import "k8s.io/api/core/v1"

type PodAdditions struct {
	Annotations    map[string]string         `json:"annotations,omitempty"`
	Labels         map[string]string         `json:"labels,omitempty"`
	Containers     []v1.Container            `json:"containers,omitempty"`
	InitContainers []v1.Container            `json:"initContainers,omitempty"`
	Volumes        []v1.Volume               `json:"volumes,omitempty"`
	PullSecrets    []v1.LocalObjectReference `json:"pullSecrets,omitempty"`
}

type WorkspaceRoutingClass string
const (
	WorkspaceRoutingDefault WorkspaceRoutingClass = ""
	WorkspaceRoutingOpenShiftOauth WorkspaceRoutingClass = "openshift-oauth"
)
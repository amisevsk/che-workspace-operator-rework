package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkspaceRoutingSpec defines the desired state of WorkspaceRouting
// +k8s:openapi-gen=true
type WorkspaceRoutingSpec struct {
	WorkspaceId         string                `json:"workspaceId"`
	RoutingClass        WorkspaceRoutingClass `json:"routingClass,omitempty"`
	IngressGlobalDomain string                `json:"ingressGlobalDomain"`
	// +listType=map +listMapKey=name
	Endpoints   []Endpoint        `json:"endpoints"`
	PodSelector map[string]string `json:"podSelector"'`
}

type WorkspaceRoutingClass string
const (
	WorkspaceRoutingDefault WorkspaceRoutingClass = ""
	WorkspaceRoutingOpenShiftOauth WorkspaceRoutingClass = "openshift-oauth"
)

// WorkspaceRoutingStatus defines the observed state of WorkspaceRouting
// +k8s:openapi-gen=true
type WorkspaceRoutingStatus struct {
	PodAdditions     *PodAdditions                `json:"podAdditions,omitempty"`
	ExposedEndpoints map[string][]ExposedEndpoint `json:"exposedEndpoints,omitempty"`
	Ready            bool                         `json:"ready"`
}

type ExposedEndpoint struct {
	Name       string                       `json:"name"`
	Url        string                       `json:"url"`
	Attributes map[EndpointAttribute]string `json:"attributes"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkspaceRouting is the Schema for the workspaceroutings API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=workspaceroutings,scope=Namespaced
type WorkspaceRouting struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkspaceRoutingSpec   `json:"spec,omitempty"`
	Status WorkspaceRoutingStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkspaceRoutingList contains a list of WorkspaceRouting
type WorkspaceRoutingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkspaceRouting `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WorkspaceRouting{}, &WorkspaceRoutingList{})
}

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ComponentSpec defines the desired state of Component
// +k8s:openapi-gen=true
type WorkspaceComponentSpec struct {
	WorkspaceId string `json:"workspaceId"`
	// +listType=map +listMapKey=name
	Components []ComponentSpec `json:"components"`
	// +listType=map +listMapKey=name
	Commands []CommandSpec `json:"commands,omitempty"`
}

// ComponentStatus defines the observed state of Component
// +k8s:openapi-gen=true
type WorkspaceComponentStatus struct {
	Ready bool `json:"ready"`
	// +listType=map +listMapKey=name
	ComponentDescriptions []ComponentDescription `json:"componentDescriptions"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Component is the Schema for the components API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=components,scope=Namespaced
type Component struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkspaceComponentSpec   `json:"spec,omitempty"`
	Status WorkspaceComponentStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ComponentList contains a list of Component
type ComponentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Component `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Component{}, &ComponentList{})
}

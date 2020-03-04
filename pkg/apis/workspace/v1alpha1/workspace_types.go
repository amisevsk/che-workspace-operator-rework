package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkspaceSpec defines the desired state of Workspace
// +k8s:openapi-gen=true
type WorkspaceSpec struct {
	// Whether the workspace should be started or stopped
	Started bool `json:"started"`
	// Routing class the defines how the workspace will be exposed to the external network
	RoutingClass string `json:"routingClass,omitempty"`
	// Workspace Structure defined in the Devfile format syntax.
	// For more details see the Che 7 documentation: https://www.eclipse.org/che/docs/che-7/making-a-workspace-portable-using-a-devfile/
	Devfile DevfileSpec `json:"devfile"`
}

// WorkspaceStatus defines the observed state of Workspace
// +k8s:openapi-gen=true
type WorkspaceStatus struct {
	WorkspaceId string `json:"workspaceId"`
	Phase WorkspacePhase `json:"phase"`
}

// WorkspacePhase is a label for the condition of a workspace at the current time.
type WorkspacePhase string

// These are the valid statuses of pods.
const (
	WorkspacePhaseStopped  WorkspacePhase = "Stopped"
	WorkspacePhaseStarting WorkspacePhase = "Starting"
	WorkspacePhaseStopping WorkspacePhase = "Stopping"
	WorkspacePhaseRunning  WorkspacePhase = "Running"
	WorkspacePhaseFailed   WorkspacePhase = "Failed"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Workspace is the Schema for the workspaces API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=workspaces,scope=Namespaced
type Workspace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkspaceSpec   `json:"spec,omitempty"`
	Status WorkspaceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkspaceList contains a list of Workspace
type WorkspaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workspace `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Workspace{}, &WorkspaceList{})
}

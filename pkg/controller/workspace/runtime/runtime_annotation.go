package runtime

import "github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"

func ConstructRuntimeAnnotation(components []v1alpha1.ComponentDescription) v1alpha1.CheWorkspaceRuntime {
	defaultEnv := "default"

	machines := getMachinesAnnotation(components)

	return v1alpha1.CheWorkspaceRuntime{
		ActiveEnv:    defaultEnv,
		Commands:     nil, // TODO
		Machines:     machines,
	}
}

func getMachinesAnnotation(components []v1alpha1.ComponentDescription) map[string]v1alpha1.CheWorkspaceMachine {
	machines := map[string]v1alpha1.CheWorkspaceMachine{}

	for _, component := range components{
		for containerName, container := range component.ComponentMetadata.Containers {
			machines[containerName] = v1alpha1.CheWorkspaceMachine{
				Attributes: container.Attributes,
				Servers:    nil, // TODO
				Status:     nil, // TODO
			}
		}
	}

	return machines
}
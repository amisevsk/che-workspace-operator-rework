package runtime

import (
	"encoding/json"
	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
)

func ConstructRuntimeAnnotation(components []v1alpha1.ComponentDescription, endpoints map[string][]v1alpha1.ExposedEndpoint) (string, error) {
	defaultEnv := "default"

	machines := getMachinesAnnotation(components, endpoints)
	commands := getWorkspaceCommands(components)

	runtime := v1alpha1.CheWorkspaceRuntime{
		ActiveEnv:    defaultEnv,
		Commands:     commands,
		Machines:     machines,
	}

	runtimeJSON, err := json.Marshal(runtime)
	if err != nil {
		return "", err
	}
	return string(runtimeJSON), nil
}

func getMachinesAnnotation(components []v1alpha1.ComponentDescription, endpoints map[string][]v1alpha1.ExposedEndpoint) map[string]v1alpha1.CheWorkspaceMachine {
	machines := map[string]v1alpha1.CheWorkspaceMachine{}

	for _, component := range components{
		for containerName, container := range component.ComponentMetadata.Containers {
			servers := map[string]v1alpha1.CheWorkspaceServer{}
			for _, endpoint := range endpoints[containerName] {
				servers[endpoint.Name] = v1alpha1.CheWorkspaceServer{
					Attributes: endpoint.Attributes, // TODO: These don't seem to map cleanly
					Status:     v1alpha1.RunningServerStatus, // TODO: This is just set so the circles are green
					URL:        endpoint.Url,
				}
			}
			machines[containerName] = v1alpha1.CheWorkspaceMachine{
				Attributes: container.Attributes,
				Servers:    servers,
			}
		}
	}

	return machines
}

func getWorkspaceCommands(components []v1alpha1.ComponentDescription) []v1alpha1.CheWorkspaceCommand {
	var commands []v1alpha1.CheWorkspaceCommand
	for _, component := range components {
		commands = append(commands, component.ComponentMetadata.ContributedRuntimeCommands...)
	}
	return commands
}
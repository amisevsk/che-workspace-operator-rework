package runtime

import (
	"encoding/json"
	"fmt"
	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
)

func ConstructRuntimeAnnotation(components []v1alpha1.ComponentDescription, endpoints map[string][]v1alpha1.ExposedEndpoint) (string, error) {
	defaultEnv := "default"

	machines := getMachinesAnnotation(components, endpoints)
	commands := getWorkspaceCommands(components)

	runtime := v1alpha1.CheWorkspaceRuntime{
		ActiveEnv: defaultEnv,
		Commands:  commands,
		Machines:  machines,
	}

	runtimeJSON, err := json.Marshal(runtime)
	if err != nil {
		return "", err
	}
	return string(runtimeJSON), nil
}

func getMachinesAnnotation(components []v1alpha1.ComponentDescription, endpoints map[string][]v1alpha1.ExposedEndpoint) map[string]v1alpha1.CheWorkspaceMachine {
	machines := map[string]v1alpha1.CheWorkspaceMachine{}

	for _, component := range components {
		for containerName, container := range component.ComponentMetadata.Containers {
			servers := map[string]v1alpha1.CheWorkspaceServer{}
			// TODO: This is likely not a good choice for matching, since it'll fail if container name does not match an endpoint key
			for _, endpoint := range endpoints[containerName] {
				protocol := endpoint.Attributes[v1alpha1.PROTOCOL_ENDPOINT_ATTRIBUTE]

				servers[endpoint.Name] = v1alpha1.CheWorkspaceServer{
					Attributes: endpoint.Attributes,
					Status:     v1alpha1.RunningServerStatus,                   // TODO: This is just set so the circles are green
					URL:        fmt.Sprintf("%s://%s", protocol, endpoint.Url), // TODO: This could potentially be done when the endpoint is created (i.e. include protocol in endpoint.Url)
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

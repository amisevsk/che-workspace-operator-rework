package provision

import (
	"context"
	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	"github.com/google/go-cmp/cmp"
)

func SyncWorkspaceStatus(workspace *v1alpha1.Workspace, exposedEndpoints map[string][]v1alpha1.ExposedEndpoint, runtime string, clusterAPI ClusterAPI) ProvisioningStatus {
	ideUrl := getIdeUrl(exposedEndpoints)
	ideUrlUpToDate := (workspace.Status.IdeUrl == ideUrl)

	runtimeUpToDate := cmp.Equal(runtime, workspace.Status.AdditionalFields.Runtime)
	if runtimeUpToDate && ideUrlUpToDate {
		return ProvisioningStatus{
			Continue: true,
		}
	}
	workspace.Status.AdditionalFields.Runtime = runtime
	workspace.Status.IdeUrl = ideUrl
	err := clusterAPI.Client.Status().Update(context.TODO(), workspace)
	return ProvisioningStatus{
		Continue: false,
		Requeue:  true,
		Err:      err,
	}
}

func getIdeUrl(exposedEndpoints map[string][]v1alpha1.ExposedEndpoint) string {
	for _, endpoints := range exposedEndpoints {
		for _, endpoint := range endpoints {
			if endpoint.Attributes[v1alpha1.TYPE_ENDPOINT_ATTRIBUTE] == "ide" {
				return endpoint.Url
			}
		}
	}
	return ""
}

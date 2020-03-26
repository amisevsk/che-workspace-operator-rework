package provision

import (
	"context"
	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	"github.com/google/go-cmp/cmp"
)

func SyncWorkspaceStatus(workspace *v1alpha1.Workspace, runtime string, clusterAPI ClusterAPI) ProvisioningStatus {
	if cmp.Equal(runtime, workspace.Status.AdditionalFields.Runtime) {
		return ProvisioningStatus{
			Continue: true,
		}
	}
	workspace.Status.AdditionalFields.Runtime = runtime
	err := clusterAPI.Client.Status().Update(context.TODO(), workspace)
	return ProvisioningStatus{
		Continue: false,
		Requeue:  true,
		Err:      err,
	}
}

package provision

import (
	"context"
	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func SyncWorkspaceStatus(workspace *v1alpha1.Workspace, runtime v1alpha1.CheWorkspaceRuntime, client client.Client) ProvisioningStatus {
	if cmp.Equal(runtime, workspace.Status.AdditionalFields.Runtime) {
		return ProvisioningStatus{
			Continue: true,
		}
	}
	workspace.Status.AdditionalFields.Runtime = runtime
	err := client.Status().Update(context.TODO(), workspace)
	return ProvisioningStatus{
		Continue: false,
		Requeue:  true,
		Err:      err,
	}
}
package provision

import (
	"context"
	"fmt"
	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type ServiceAcctProvisioningStatus struct {
	ProvisioningStatus
	ServiceAccountName string
}

func SyncServiceAccount(
	workspace *v1alpha1.Workspace,
	additionalAnnotations map[string]string,
	clusterAPI ClusterAPI) ServiceAcctProvisioningStatus {
	// note: autoMountServiceAccount := true comes from a hardcoded value in prerequisites.go
	autoMountServiceAccount := true
	saName := "che-" + workspace.Status.WorkspaceId

	specSA := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: workspace.Namespace,
		},
		AutomountServiceAccountToken: &autoMountServiceAccount,
	}

	if len(additionalAnnotations) > 0 {
		specSA.Annotations = map[string]string{}
		for annotKey, annotVal := range additionalAnnotations {
			specSA.Annotations[annotKey] = annotVal
		}
	}

	err := controllerutil.SetControllerReference(workspace, &specSA, clusterAPI.Scheme)
	if err != nil {
		return ServiceAcctProvisioningStatus{
			ProvisioningStatus: ProvisioningStatus{
				Err: err,
			},
		}
	}
	clusterSA, err := getClusterSA(specSA, clusterAPI.Client)
	if err != nil {
		return ServiceAcctProvisioningStatus{
			ProvisioningStatus: ProvisioningStatus{
				Err: err,
			},
		}
	}

	if clusterSA == nil {
		clusterAPI.Logger.Info("Creating workspace ServiceAccount")
		err := clusterAPI.Client.Create(context.TODO(), &specSA)
		return ServiceAcctProvisioningStatus{
			ProvisioningStatus: ProvisioningStatus{
				Continue: false,
				Requeue:  true,
				Err:      err,
			},
		}
	}

	if !cmp.Equal(specSA.Annotations, clusterSA.Annotations) {
		clusterAPI.Logger.Info("Updating workspace ServiceAccount")
		fmt.Printf("\n\n%s\n\n", cmp.Diff(specSA, *clusterSA))
		patch := runtimeClient.MergeFrom(&specSA)
		err := clusterAPI.Client.Patch(context.TODO(), clusterSA, patch)
		if err != nil {
			if errors.IsConflict(err) {
				return ServiceAcctProvisioningStatus{ProvisioningStatus: ProvisioningStatus{Requeue: true}}
			}
			return ServiceAcctProvisioningStatus{ProvisioningStatus: ProvisioningStatus{Err: err}}
		}
		return ServiceAcctProvisioningStatus{
			ProvisioningStatus: ProvisioningStatus{Requeue: true},
		}
	}

	return ServiceAcctProvisioningStatus{
		ProvisioningStatus: ProvisioningStatus{
			Continue: true,
		},
		ServiceAccountName: saName,
	}
}

func getClusterSA(sa corev1.ServiceAccount, client runtimeClient.Client) (*corev1.ServiceAccount, error) {
	clusterSA := &corev1.ServiceAccount{}
	namespacedName := types.NamespacedName{
		Name:      sa.Name,
		Namespace: sa.Namespace,
	}
	err := client.Get(context.TODO(), namespacedName, clusterSA)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return clusterSA, nil
}

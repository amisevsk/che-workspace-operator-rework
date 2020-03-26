package provision

import (
	"context"
	"fmt"
	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	config2 "github.com/che-incubator/che-workspace-operator/pkg/config"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type RoutingProvisioningStatus struct {
	ProvisioningStatus
	PodAdditions     *v1alpha1.PodAdditions
	ExposedEndpoints map[string][]v1alpha1.ExposedEndpoint
}

var routingDiffOpts = cmp.Options{
	cmpopts.IgnoreFields(v1alpha1.WorkspaceRouting{}, "TypeMeta", "ObjectMeta", "Status"),
}

func SyncRoutingToCluster(
	workspace *v1alpha1.Workspace,
	components []v1alpha1.ComponentDescription,
	clusterAPI ClusterAPI) RoutingProvisioningStatus {

	specRouting, err := getSpecRouting(workspace, components, clusterAPI.Scheme)
	if err != nil {
		return RoutingProvisioningStatus{
			ProvisioningStatus: ProvisioningStatus{Err: err},
		}
	}

	clusterRouting, err := getClusterRouting(specRouting.Name, specRouting.Namespace, clusterAPI.Client)
	if err != nil {
		return RoutingProvisioningStatus{
			ProvisioningStatus: ProvisioningStatus{Err: err},
		}
	}

	if clusterRouting == nil {
		err := clusterAPI.Client.Create(context.TODO(), specRouting)
		return RoutingProvisioningStatus{
			ProvisioningStatus: ProvisioningStatus{Requeue: true, Err: err},
		}
	}

	if !cmp.Equal(specRouting, clusterRouting, routingDiffOpts) {
		clusterRouting.Spec = specRouting.Spec
		err := clusterAPI.Client.Update(context.TODO(), clusterRouting)
		return RoutingProvisioningStatus{
			ProvisioningStatus: ProvisioningStatus{Requeue: true, Err: err},
		}
	}

	if !clusterRouting.Status.Ready {
		return RoutingProvisioningStatus{
			ProvisioningStatus: ProvisioningStatus{
				Continue: false,
				Requeue:  false,
			},
		}
	}

	return RoutingProvisioningStatus{
		ProvisioningStatus: ProvisioningStatus{
			Continue: clusterRouting.Status.Ready,
		},
		PodAdditions:     clusterRouting.Status.PodAdditions,
		ExposedEndpoints: clusterRouting.Status.ExposedEndpoints,
	}
}

func getSpecRouting(
	workspace *v1alpha1.Workspace,
	componentDescriptions []v1alpha1.ComponentDescription,
	scheme *runtime.Scheme) (*v1alpha1.WorkspaceRouting, error) {

	endpoints := map[string][]v1alpha1.Endpoint{}
	for _, desc := range componentDescriptions {
		endpoints[desc.Name] = append(endpoints[desc.Name], desc.ComponentMetadata.Endpoints...)
	}

	routing := &v1alpha1.WorkspaceRouting{
		ObjectMeta: v1.ObjectMeta{
			Name:      fmt.Sprintf("routing-%s", workspace.Status.WorkspaceId),
			Namespace: workspace.Namespace,
		},
		Spec: v1alpha1.WorkspaceRoutingSpec{
			WorkspaceId:         workspace.Status.WorkspaceId,
			RoutingClass:        workspace.Spec.RoutingClass,
			IngressGlobalDomain: config2.ControllerCfg.GetIngressGlobalDomain(),
			Endpoints:           endpoints,
			PodSelector: map[string]string{
				"app": workspace.Status.WorkspaceId,
			},
		},
	}
	err := controllerutil.SetControllerReference(workspace, routing, scheme)
	if err != nil {
		return nil, err
	}

	return routing, nil
}

func getClusterRouting(name string, namespace string, client runtimeClient.Client) (*v1alpha1.WorkspaceRouting, error) {
	routing := &v1alpha1.WorkspaceRouting{}
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	err := client.Get(context.TODO(), namespacedName, routing)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return routing, nil
}

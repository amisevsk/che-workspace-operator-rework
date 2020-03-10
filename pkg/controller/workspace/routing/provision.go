package routing

import (
	"context"
	"fmt"
	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	"github.com/che-incubator/che-workspace-operator/pkg/controller/workspace/common"
	"github.com/che-incubator/che-workspace-operator/pkg/controller/workspace/config"
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
	common.ProvisioningStatus
	PodAdditions     *v1alpha1.PodAdditions
	ExposedEndpoints map[string][]v1alpha1.ExposedEndpoint
}

var routingDiffOpts = cmp.Options{
	cmpopts.IgnoreFields(v1alpha1.WorkspaceRouting{}, "TypeMeta", "ObjectMeta", "Status"),
}

func SyncObjectsToCluster(
		workspace *v1alpha1.Workspace,
		components []v1alpha1.ComponentDescription,
		client runtimeClient.Client,
		scheme *runtime.Scheme) RoutingProvisioningStatus {

	specRouting, err := getSpecRouting(workspace, components, scheme)
	if err != nil {
		return RoutingProvisioningStatus{
			ProvisioningStatus: common.ProvisioningStatus{Err: err},
		}
	}

	clusterRouting, err := getClusterRouting(specRouting.Name, specRouting.Namespace, client)
	if err != nil {
		return RoutingProvisioningStatus{
			ProvisioningStatus: common.ProvisioningStatus{Err: err},
		}
	}

	if clusterRouting == nil {
		err := client.Create(context.TODO(), specRouting)
		return RoutingProvisioningStatus{
			ProvisioningStatus: common.ProvisioningStatus{Requeue: true, Err: err},
		}
	}

	if !cmp.Equal(specRouting, clusterRouting, routingDiffOpts) {
		clusterRouting.Spec = specRouting.Spec
		err := client.Update(context.TODO(), clusterRouting)
		return RoutingProvisioningStatus{
			ProvisioningStatus: common.ProvisioningStatus{Requeue: true, Err: err},
		}
	}

	if !clusterRouting.Status.Ready {
		return RoutingProvisioningStatus{
			ProvisioningStatus: common.ProvisioningStatus{
				Continue:              false,
				Requeue:               false,
			},
		}
	}

	return RoutingProvisioningStatus{
		ProvisioningStatus: common.ProvisioningStatus{
			Continue:              clusterRouting.Status.Ready,
		},
		PodAdditions:       clusterRouting.Status.PodAdditions,
		ExposedEndpoints:   clusterRouting.Status.ExposedEndpoints,
	}
}

func getSpecRouting(
		workspace *v1alpha1.Workspace,
		componentDescriptions []v1alpha1.ComponentDescription,
		scheme *runtime.Scheme) (*v1alpha1.WorkspaceRouting, error) {

	var endpoints []v1alpha1.Endpoint
	for _, desc := range componentDescriptions {
		endpoints = append(endpoints, desc.ComponentMetadata.Endpoints...)
	}

	routing := &v1alpha1.WorkspaceRouting{
		ObjectMeta: v1.ObjectMeta{
			Name:      fmt.Sprintf("routing-%s", workspace.Status.WorkspaceId),
			Namespace: workspace.Namespace,
		},
		Spec: v1alpha1.WorkspaceRoutingSpec{
			WorkspaceId:         workspace.Status.WorkspaceId,
			RoutingClass:        workspace.Spec.RoutingClass,
			IngressGlobalDomain: config.ControllerCfg.GetIngressGlobalDomain(),
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

package workspace

import (
	"context"
	"fmt"
	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	"github.com/che-incubator/che-workspace-operator/pkg/controller/workspace/config"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var routingDiffOpts = cmp.Options{
	cmpopts.IgnoreFields(v1alpha1.WorkspaceRouting{}, "TypeMeta", "ObjectMeta", "Status"),
}

func (r *ReconcileWorkspace) getSpecRouting(
		workspace *v1alpha1.Workspace,
		componentDescriptions []v1alpha1.ComponentDescription) (*v1alpha1.WorkspaceRouting, error) {

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
	err := controllerutil.SetControllerReference(workspace, routing, r.scheme)
	if err != nil {
		return nil, err
	}

	return routing, nil
}

func (r *ReconcileWorkspace) getClusterRouting(name string, namespace string) (*v1alpha1.WorkspaceRouting, error) {
	routing := &v1alpha1.WorkspaceRouting{}
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	err := r.client.Get(context.TODO(), namespacedName, routing)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return routing, nil
}

func diffRouting(specRouting, clusterRouting *v1alpha1.WorkspaceRouting) bool {
	return cmp.Equal(specRouting, clusterRouting, routingDiffOpts)
}

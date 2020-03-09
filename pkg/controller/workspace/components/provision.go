package components

import (
	"context"
	"fmt"
	"github.com/che-incubator/che-workspace-operator/pkg/adaptor"
	v1alpha1 "github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	workspaceAPI "github.com/che-incubator/che-workspace-operator/pkg/controller/workspace"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type ComponentsProvisioningPhase struct{}

var _ workspaceAPI.ProvisioningPhase = (*ComponentsProvisioningPhase)(nil)

var componentDiffOpts = cmp.Options{
	cmpopts.IgnoreFields(v1alpha1.Component{}, "TypeMeta", "ObjectMeta", "Status"),
}

func (p *ComponentsProvisioningPhase) SyncObjectsToCluster(
		workspace *v1alpha1.Workspace, client runtimeClient.Client, scheme *runtime.Scheme) workspaceAPI.ProvisioningStatus {
	specComponents, err := getSpecComponents(workspace, scheme)
	if err != nil {
		return workspaceAPI.ProvisioningStatus{
			Err: err,
		}
	}

	clusterComponents, err := getClusterComponents(workspace, client)
	if err != nil {
		return workspaceAPI.ProvisioningStatus{
			Err: err,
		}
	}

	toCreate, toUpdate, toDelete := sortComponents(specComponents, clusterComponents)
	if len(toCreate) == 0 && len(toUpdate) == 0 && len(toDelete) == 0 {
		var componentDescriptions []v1alpha1.ComponentDescription
		for _, component := range clusterComponents {
			componentDescriptions = append(componentDescriptions, component.Status.ComponentDescriptions...)
		}
		return workspaceAPI.ProvisioningStatus{
			Continue:              true,
			ComponentDescriptions: componentDescriptions,
		}
	}

	for _, component := range toCreate {
		err := client.Create(context.TODO(), &component)
		if err != nil {
			return workspaceAPI.ProvisioningStatus{
				Err: err,
			}
		}
	}

	for _, component := range toUpdate {
		err := client.Update(context.TODO(), &component)
		if err != nil {
			return workspaceAPI.ProvisioningStatus{
				Err: err,
			}
		}
	}

	for _, component := range toDelete {
		err := client.Delete(context.TODO(), &component)
		if err != nil {
			return workspaceAPI.ProvisioningStatus{
				Err: err,
			}
		}
	}

	return workspaceAPI.ProvisioningStatus{
		Continue:              false,
		Requeue:               true,
	}
}

func getSpecComponents(workspace *v1alpha1.Workspace, scheme *runtime.Scheme) ([]v1alpha1.Component, error) {
	dockerComponents, pluginComponents, err := adaptor.SortComponentsByType(workspace.Spec.Devfile.Components)
	if err != nil {
		return nil, err
	}

	dockerResolver := v1alpha1.Component{
		ObjectMeta: v1.ObjectMeta{
			Name:      fmt.Sprintf("components-%s-%s", workspace.Status.WorkspaceId, "docker"),
			Namespace: workspace.Namespace,
			Labels: map[string]string{
				"app": workspace.Status.WorkspaceId,
			},
		},
		Spec: v1alpha1.WorkspaceComponentSpec{
			WorkspaceId: workspace.Status.WorkspaceId,
			Components:  dockerComponents,
		},
	}
	pluginResolver := v1alpha1.Component{
		ObjectMeta: v1.ObjectMeta{
			Name:      fmt.Sprintf("components-%s-%s", workspace.Status.WorkspaceId, "plugins"),
			Namespace: workspace.Namespace,
			Labels: map[string]string{
				"app": workspace.Status.WorkspaceId,
			},
		},
		Spec: v1alpha1.WorkspaceComponentSpec{
			WorkspaceId: workspace.Status.WorkspaceId,
			Components:  pluginComponents,
		},
	}
	controllerutil.SetControllerReference(workspace, &dockerResolver, scheme)
	controllerutil.SetControllerReference(workspace, &pluginResolver, scheme)

	return []v1alpha1.Component{pluginResolver, dockerResolver}, nil
}

func getClusterComponents(workspace *v1alpha1.Workspace, client runtimeClient.Client) ([]v1alpha1.Component, error) {
	found := &v1alpha1.ComponentList{}
	labelSelector, err := labels.Parse(fmt.Sprintf("app=%s", workspace.Status.WorkspaceId))
	if err != nil {
		return nil, err
	}
	listOptions := &runtimeClient.ListOptions{
		Namespace:     workspace.Namespace,
		LabelSelector: labelSelector,
	}
	err = client.List(context.TODO(), found, listOptions)
	if err != nil {
		return nil, err
	}
	return found.Items, nil
}

func sortComponents(spec, cluster []v1alpha1.Component) (create, update, delete []v1alpha1.Component) {
	for _, clusterComponent := range cluster {
		if contains, _ := listContainsByName(clusterComponent, spec); !contains {
			delete = append(delete, clusterComponent)
		}
	}

	for _, specComponent := range spec {
		if contains, idx := listContainsByName(specComponent, cluster); contains {
			clusterComponent := cluster[idx]
			if !cmp.Equal(specComponent, clusterComponent, componentDiffOpts) {
				clusterComponent.Spec = specComponent.Spec
				update = append(update, clusterComponent)
			}
		} else {
			create = append(create, specComponent)
		}
	}
}

func listContainsByName(query v1alpha1.Component, list []v1alpha1.Component) (exists bool, idx int) {
	for idx, listItem := range list {
		if query.Name == listItem.Name {
			return true, idx
		}
	}
	return false, -1
}

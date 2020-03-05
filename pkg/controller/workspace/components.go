package workspace

import (
	"context"
	"fmt"
	"github.com/che-incubator/che-workspace-operator/pkg/adaptor"
	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var componentDiffOpts = cmp.Options{
	cmpopts.IgnoreFields(v1alpha1.Component{}, "TypeMeta", "ObjectMeta", "Status"),
}

func (r *ReconcileWorkspace) syncComponents(specComponents, clusterComponents []v1alpha1.Component) error {
	create, update, delete := sortComponents(specComponents, clusterComponents)
	for _, toCreate := range create {
		log.Info("Creating " + toCreate.Name)
		err := r.client.Create(context.TODO(), &toCreate)
		if err != nil {
			return err
		}
	}
	for _, toUpdate := range update {
		log.Info("Updating " + toUpdate.Name)
		err := r.client.Update(context.TODO(), &toUpdate)
		if err != nil {
			return err
		}
	}
	for _, toDelete := range delete {
		log.Info("Deleting " + toDelete.Name)
		err := r.client.Delete(context.TODO(), &toDelete)
		if err != nil {
			return err
		}
	}
	return nil
}

func sortComponents(spec, cluster []v1alpha1.Component) (create, update, delete []v1alpha1.Component) {
	for _, specComponent := range spec {
		if contains, _ := containsComponent(cluster, specComponent); !contains {
			create = append(create, specComponent)
		}
	}
	for _, clusterComponent := range cluster {
		if contains, match := containsComponent(spec, clusterComponent); contains {
			clusterComponent.Spec = match.Spec
			update = append(update, clusterComponent)
		} else {
			delete = append(delete, clusterComponent)
		}
	}
	return
}

func containsComponent(list []v1alpha1.Component, query v1alpha1.Component) (bool, *v1alpha1.Component) {
	for _, item := range list {
		if item.Name == query.Name {
			return true, &item
		}
	}
	return false, nil
}

func (r *ReconcileWorkspace) checkComponents(specComponents, clusterComponents []v1alpha1.Component) (exist, ready bool, msg string) {

	if len(specComponents) != len(clusterComponents) {
		return false, false, "Components do not match"
	}
	for _, specComponent := range specComponents {
		if !componentExistsInList(specComponent, clusterComponents) {
			return false, false, fmt.Sprintf("Component %s needs to be created or updated", specComponent.Name)
		}
	}
	for _, clusterComponent := range clusterComponents {
		if !componentExistsInList(clusterComponent, specComponents) {
			return false, false, fmt.Sprintf("Component %s needs to be deleted from cluster", clusterComponent.Name)
		}
	}

	for _, clusterComponent := range clusterComponents {
		if !clusterComponent.Status.Ready {
			return true, false, fmt.Sprintf("Component %s not ready", clusterComponent.Name)
		}
	}

	return true, true, ""
}

func componentExistsInList(component v1alpha1.Component, componentsList []v1alpha1.Component) bool {
	for _, listComponent := range componentsList {
		if cmp.Equal(component, listComponent, componentDiffOpts) {
			return true
		}
	}
	return false
}

func (r *ReconcileWorkspace) getSpecComponents(workspace *v1alpha1.Workspace) ([]v1alpha1.Component, error) {
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
	controllerutil.SetControllerReference(workspace, &dockerResolver, r.scheme)
	controllerutil.SetControllerReference(workspace, &pluginResolver, r.scheme)

	return []v1alpha1.Component{pluginResolver, dockerResolver}, nil
}

func (r *ReconcileWorkspace) getClusterComponents(workspace *v1alpha1.Workspace) ([]v1alpha1.Component, error) {
	found := &v1alpha1.ComponentList{}
	labelSelector, err := labels.Parse(fmt.Sprintf("app=%s", workspace.Status.WorkspaceId))
	if err != nil {
		return nil, err
	}
	listOptions := &client.ListOptions{
		Namespace:     workspace.Namespace,
		LabelSelector: labelSelector,
	}
	err = r.client.List(context.TODO(), found, listOptions)
	if err != nil {
		return nil, err
	}
	return found.Items, nil
}

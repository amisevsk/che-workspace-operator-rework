package workspacerouting

import (
	"context"
	"fmt"
	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	"github.com/che-incubator/che-workspace-operator/pkg/config"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ingressDiffOpts = cmp.Options{
	cmpopts.IgnoreFields(v1beta1.Ingress{}, "TypeMeta", "ObjectMeta", "Status"),
}

func (r *ReconcileWorkspaceRouting) syncIngresses(routing *v1alpha1.WorkspaceRouting, specIngresses []v1beta1.Ingress) (ok bool, err error) {
	ingressesInSync := true

	clusterIngresses, err := r.getClusterIngresses(routing)
	if err != nil {
		return false, err
	}

	toDelete := getIngressesToDelete(clusterIngresses, specIngresses)
	for _, ingress := range toDelete {
		err := r.client.Delete(context.TODO(), &ingress)
		if err != nil {
			return false, err
		}
		ingressesInSync = false
	}

	for _, specIngress := range specIngresses {
		if contains, idx := listContainsIngressByName(specIngress, clusterIngresses); contains {
			clusterIngress := clusterIngresses[idx]
			if !cmp.Equal(specIngress, clusterIngress, ingressDiffOpts) {
				// Update ingress's spec
				clusterIngress.Spec = specIngress.Spec
				err := r.client.Update(context.TODO(), &clusterIngress)
				if err != nil && !errors.IsConflict(err) {
					return false, err
				}
				ingressesInSync = false
			}
		} else {
			err := r.client.Create(context.TODO(), &specIngress)
			if err != nil {
				return false, err
			}
			ingressesInSync = false
		}
	}

	return ingressesInSync, nil
}

func (r *ReconcileWorkspaceRouting) getClusterIngresses(routing *v1alpha1.WorkspaceRouting) ([]v1beta1.Ingress, error) {
	found := &v1beta1.IngressList{}
	labelSelector, err := labels.Parse(fmt.Sprintf("%s=%s", config.WorkspaceIDLabel, routing.Spec.WorkspaceId))
	if err != nil {
		return nil, err
	}
	listOptions := &client.ListOptions{
		Namespace:     routing.Namespace,
		LabelSelector: labelSelector,
	}
	err = r.client.List(context.TODO(), found, listOptions)
	if err != nil {
		return nil, err
	}
	return found.Items, nil
}

func getIngressesToDelete(clusterIngresses, specIngresses []v1beta1.Ingress) []v1beta1.Ingress {
	var toDelete []v1beta1.Ingress
	for _, clusterIngress := range clusterIngresses {
		if contains, _ := listContainsIngressByName(clusterIngress, specIngresses); !contains {
			toDelete = append(toDelete, clusterIngress)
		}
	}
	return toDelete
}

func listContainsIngressByName(query v1beta1.Ingress, list []v1beta1.Ingress) (exists bool, idx int) {
	for idx, listIngress := range list {
		if query.Name == listIngress.Name {
			return true, idx
		}
	}
	return false, -1
}

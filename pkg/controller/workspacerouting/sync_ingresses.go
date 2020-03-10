package workspacerouting

import (
	"context"
	"fmt"
	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ingressDiffOpts = cmp.Options{
	cmpopts.IgnoreFields(v1beta1.Ingress{}, "TypeMeta", "ObjectMeta", "Status"),
}

func (r *ReconcileWorkspaceRouting) syncIngresses(routing *v1alpha1.WorkspaceRouting, specIngresses []v1beta1.Ingress) (ok bool, err error) {
	servicesInSync := true

	clusterIngresses, err := r.getClusterIngresses(routing)
	if err != nil {
		return false, err
	}

	toDelete := getIngressesToDelete(clusterIngresses, specIngresses)
	for _, service := range toDelete {
		err := r.client.Delete(context.TODO(), &service)
		if err != nil {
			return false, err
		}
		servicesInSync = false
	}

	for _, specIngress := range specIngresses {
		if contains, idx := listContainsIngressByName(specIngress, clusterIngresses); contains {
			clusterService := clusterIngresses[idx]
			if !cmp.Equal(specIngress, clusterService, ingressDiffOpts) {
				fmt.Printf("\n\n%s\n\n", cmp.Diff(specIngress, clusterIngresses, ingressDiffOpts))
				// Update service's spec
				clusterService.Spec = specIngress.Spec
				err := r.client.Update(context.TODO(), &clusterService)
				if err != nil {
					return false, err
				}
				servicesInSync = false
			}
		} else {
			err := r.client.Create(context.TODO(), &specIngress)
			if err != nil {
				return false, err
			}
			servicesInSync = false
		}
	}

	return servicesInSync, nil
}

func (r *ReconcileWorkspaceRouting) getClusterIngresses(routing *v1alpha1.WorkspaceRouting) ([]v1beta1.Ingress, error) {
	found := &v1beta1.IngressList{}
	labelSelector, err := labels.Parse(fmt.Sprintf("app=%s", routing.Spec.WorkspaceId)) // TODO This is manually synced with what's created, that's bad.
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

func getIngressesToDelete(clusterIngresses, specServices []v1beta1.Ingress) []v1beta1.Ingress {
	var toDelete []v1beta1.Ingress
	for _, clusterIngress := range clusterIngresses {
		if contains, _ := listContainsIngressByName(clusterIngress, specServices); !contains {
			toDelete = append(toDelete, clusterIngress)
		}
	}
	return toDelete
}

func listContainsIngressByName(query v1beta1.Ingress, list []v1beta1.Ingress) (exists bool, idx int) {
	for idx, listService := range list {
		if query.Name == listService.Name {
			return true, idx
		}
	}
	return false, -1
}

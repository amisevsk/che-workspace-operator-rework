package workspacerouting

import (
	"context"
	"fmt"
	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	routeV1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ReconcileWorkspaceRouting) getClusterServices(routing v1alpha1.WorkspaceRouting) ([]corev1.Service, error) {
	found := &corev1.ServiceList{}
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

func (r *ReconcileWorkspaceRouting) getClusterIngresses(routing v1alpha1.WorkspaceRouting) ([]v1beta1.Ingress, error) {
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

func (r *ReconcileWorkspaceRouting) getClusterRoutes(routing v1alpha1.WorkspaceRouting) ([]routeV1.Route, error) {
	found := &routeV1.RouteList{}
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

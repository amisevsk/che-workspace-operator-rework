package solvers

import (
	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	v12 "github.com/openshift/api/route/v1"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
)

type RoutingObjects struct {
	Services         []v1.Service
	Ingresses        []v1beta1.Ingress
	Routes           []v12.Route
	PodAdditions     *v1alpha1.PodAdditions
	ExposedEndpoints map[string][]v1alpha1.ExposedEndpoint
}

type RoutingSolver interface {
	GetSpecObjects(spec v1alpha1.WorkspaceRoutingSpec, workspaceMeta WorkspaceMetadata) RoutingObjects
}

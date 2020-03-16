package solvers

import (
	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
)

var ingressAnnotations = map[string]string{
	"kubernetes.io/ingress.class":                "nginx",
	"nginx.ingress.kubernetes.io/rewrite-target": "/",
	"nginx.ingress.kubernetes.io/ssl-redirect":   "false",
}

type BasicSolver struct {}

var _ RoutingSolver = (*BasicSolver)(nil)

func (s *BasicSolver) GetSpecObjects(spec v1alpha1.WorkspaceRoutingSpec, workspaceMeta WorkspaceMetadata) RoutingObjects {
	services := getServicesForEndpoints(spec.Endpoints, workspaceMeta)
	ingresses, exposedEndpoints := getIngressesForSpec(spec.Endpoints, workspaceMeta)

	return RoutingObjects{
		Services: services,
		Ingresses: ingresses,
		ExposedEndpoints: exposedEndpoints,
	}
}


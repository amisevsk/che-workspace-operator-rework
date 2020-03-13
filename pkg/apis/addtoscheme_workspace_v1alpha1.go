package apis

import (
	"github.com/che-incubator/che-workspace-operator/internal/cluster"
	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	routeV1 "github.com/openshift/api/route/v1"
	templateV1 "github.com/openshift/api/template/v1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, v1alpha1.SchemeBuilder.AddToScheme)
	if isOS, err := cluster.IsOpenShift(); isOS && err == nil {
		AddToSchemes = append(AddToSchemes,
			routeV1.AddToScheme,
			templateV1.AddToScheme,
		)
	}
}

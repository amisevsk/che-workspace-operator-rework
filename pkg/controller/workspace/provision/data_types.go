package provision

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ProvisioningStatus struct {
	// Continue should be true if cluster state matches spec state for this step
	Continue bool
	Requeue  bool
	Err      error
}

type ClusterAPI struct {
	Client client.Client
	Scheme *runtime.Scheme
	Logger logr.Logger
}

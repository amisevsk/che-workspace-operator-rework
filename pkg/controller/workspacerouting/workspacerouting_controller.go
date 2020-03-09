package workspacerouting

import (
	"context"
	"github.com/che-incubator/che-workspace-operator/internal/cluster"
	workspacev1alpha1 "github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	routeV1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_workspacerouting")

type Solver interface {
	SyncRoutingObjects()
}

// Add creates a new WorkspaceRouting Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileWorkspaceRouting{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("workspacerouting-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource WorkspaceRouting
	err = c.Watch(&source.Kind{Type: &workspacev1alpha1.WorkspaceRouting{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resources: Services, Ingresses, and (on OpenShift) Routes.
	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &workspacev1alpha1.WorkspaceRouting{},
	})
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &v1beta1.Ingress{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &workspacev1alpha1.WorkspaceRouting{},
	})
	if err != nil {
		return err
	}

	isOpenShift, err := cluster.IsOpenShift()
	if err != nil {
		log.Error(err, "Failed to determine if running in OpenShift")
		return err
	}
	if isOpenShift {
		err = c.Watch(&source.Kind{Type: &routeV1.Route{}}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &workspacev1alpha1.WorkspaceRouting{},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// blank assignment to verify that ReconcileWorkspaceRouting implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileWorkspaceRouting{}

// ReconcileWorkspaceRouting reconciles a WorkspaceRouting object
type ReconcileWorkspaceRouting struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a WorkspaceRouting object and makes changes based on the state read
// and what is in the WorkspaceRouting.Spec
func (r *ReconcileWorkspaceRouting) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling WorkspaceRouting")

	// Fetch the WorkspaceRouting instance
	instance := &workspacev1alpha1.WorkspaceRouting{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	services, ingresses, _ := GetSpecObjects(instance.Spec, instance.Namespace)
	for _, service := range services {
		controllerutil.SetControllerReference(instance, &service, r.scheme)
	}
	for _, ingress := range ingresses {
		controllerutil.SetControllerReference(instance, &ingress, r.scheme)
	}

	servicesInSync, err := r.syncServices(instance, services)
	if err != nil || !servicesInSync {
		return reconcile.Result{Requeue: true}, err
	}

	ingressesInSync, err := r.syncIngresses(instance, ingresses)
	if err != nil || !ingressesInSync {
		return reconcile.Result{Requeue: true}, err
	}

	instance.Status.Ready = true
	instance.Status.PodAdditions = nil
	instance.Status.ExposedEndpoints = nil // TODO
	err = r.client.Status().Update(context.TODO(), instance)
	return reconcile.Result{}, err
}

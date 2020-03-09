package workspace

import (
	"context"
	"fmt"
	workspacev1alpha1 "github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	"github.com/che-incubator/che-workspace-operator/pkg/controller/workspace/config"
	"github.com/google/uuid"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
)

var log = logf.Log.WithName("controller_workspace")

// Add creates a new Workspace Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileWorkspace{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("workspace-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	operatorNamespace, err := k8sutil.GetOperatorNamespace()
	if err == nil {
		config.ConfigMapReference.Namespace = operatorNamespace
	} else if err == k8sutil.ErrRunLocal {
		config.ConfigMapReference.Namespace = os.Getenv("WATCH_NAMESPACE")
		log.Info(fmt.Sprintf("Running operator in local mode; watching namespace %s", config.ConfigMapReference.Namespace))
	} else if err != k8sutil.ErrNoNamespace {
		return err
	}

	err = config.WatchControllerConfig(c, mgr)
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Workspace
	err = c.Watch(&source.Kind{Type: &workspacev1alpha1.Workspace{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Deployments and requeue the owner Workspace
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &workspacev1alpha1.Workspace{},
	})
	if err != nil {
		return err
	}

	// Watch for changes in secondary resource Components and requeue the owner workspace
	err = c.Watch(&source.Kind{Type: &workspacev1alpha1.Component{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &workspacev1alpha1.Workspace{},
	})

	// TODO: Watch workspaceroutings as well later

	return nil
}

// blank assignment to verify that ReconcileWorkspace implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileWorkspace{}

// ReconcileWorkspace reconciles a Workspace object
type ReconcileWorkspace struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Workspace object and makes changes based on the state read
// and what is in the Workspace.Spec
func (r *ReconcileWorkspace) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Workspace")

	// Fetch the Workspace instance
	workspace := &workspacev1alpha1.Workspace{}
	err := r.client.Get(context.TODO(), request.NamespacedName, workspace)
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

	// Ensure workspaceID is set.
	if workspace.Status.WorkspaceId == "" {
		workspaceId, err := getWorkspaceId(workspace)
		if err != nil {
			return reconcile.Result{}, err
		}
		workspace.Status.WorkspaceId = workspaceId
	}

	// Get list of components we expect from the spec
	specComponents, err := r.getSpecComponents(workspace)
	if err != nil {
		return reconcile.Result{}, err
	}
	// Get currently deployed components
	clusterComponents, err := r.getClusterComponents(workspace)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Check that the created components match the current spec,
	exist, ready, msg := r.checkComponents(specComponents, clusterComponents)
	if !exist {
		reqLogger.Info("Creating components; info: " + msg)
		createErr := r.syncComponents(specComponents, clusterComponents)
		if createErr != nil {
			return reconcile.Result{}, createErr
		}
		workspace.Status.Status = workspacev1alpha1.WorkspaceStatusStarting
		updateErr := r.client.Status().Update(context.TODO(), workspace)
		return reconcile.Result{Requeue: true}, updateErr
	}

	if !ready {
		return reconcile.Result{}, nil
	}

	if workspace.Status.Status != workspacev1alpha1.WorkspaceStatusStarted {
		workspace.Status.Status = workspacev1alpha1.WorkspaceStatusStarted
		updateErr := r.client.Status().Update(context.TODO(), workspace)
		return reconcile.Result{Requeue: true}, updateErr
	}

	var componentDescriptions []workspacev1alpha1.ComponentDescription
	for _, clusterComponent := range clusterComponents {
		componentDescriptions = append(componentDescriptions, clusterComponent.Status.ComponentDescriptions...)
	}

	specRouting, err := r.getSpecRouting(workspace, componentDescriptions)
	if err != nil {
		return reconcile.Result{}, err
	}
	clusterRouting, err := r.getClusterRouting(specRouting.Name, specRouting.Namespace)
	if err != nil {
		return reconcile.Result{}, err
	}
	if clusterRouting == nil {
		reqLogger.Info("Creating WorkspaceRouting")
		err := r.client.Create(context.TODO(), specRouting)
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	if !diffRouting(specRouting, clusterRouting) {
		reqLogger.Info("Updating WorkspaceRouting")
		clusterRouting.Spec = specRouting.Spec
		err := r.client.Update(context.TODO(), clusterRouting)
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	if !clusterRouting.Status.Ready {
		reqLogger.Info("Waiting on WorkspaceRouting")
		return reconcile.Result{}, nil
	}

	var podAdditions []workspacev1alpha1.PodAdditions
	for _, componentDesc := range componentDescriptions {
		podAdditions = append(podAdditions, componentDesc.PodAdditions)
	}
	if clusterRouting.Status.PodAdditions != nil {
		podAdditions = append(podAdditions, *clusterRouting.Status.PodAdditions)
	}

	workspaceDeployment, err := r.createWorkspaceDeployment(workspace, podAdditions)
	if err != nil {
		return reconcile.Result{}, err
	}

	reqLogger.Info("Creating workspace deployment")
	err = r.client.Create(context.TODO(), workspaceDeployment)
	if err != nil {
		return reconcile.Result{}, err
	}

	reqLogger.Info("Everything ready :)")
	return reconcile.Result{}, nil
}

func getWorkspaceId(instance *workspacev1alpha1.Workspace) (string, error) {
	uid, err := uuid.Parse(string(instance.UID))
	if err != nil {
		return "", err
	}
	return "workspace" + strings.Join(strings.Split(uid.String(), "-")[0:3], ""), nil
}

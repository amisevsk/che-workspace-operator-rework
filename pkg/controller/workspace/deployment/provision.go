package deployment

import (
	"context"
	"fmt"
	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	"github.com/che-incubator/che-workspace-operator/pkg/controller/workspace/common"
	"github.com/che-incubator/che-workspace-operator/pkg/controller/workspace/config"
	"github.com/che-incubator/che-workspace-operator/pkg/controller/workspace/model"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type DeploymentProvisioningStatus struct {
	common.ProvisioningStatus
	Status string
}

var deploymentDiffOpts = cmp.Options{
	cmpopts.IgnoreFields(appsv1.Deployment{}, "TypeMeta", "ObjectMeta", "Status"),
	cmpopts.IgnoreFields(appsv1.DeploymentSpec{}, "RevisionHistoryLimit", "ProgressDeadlineSeconds"),
	cmpopts.IgnoreFields(corev1.PodSpec{}, "DNSPolicy", "SchedulerName"),
	// TODO: Should we really be ignoring pullPolicy?
	cmpopts.IgnoreFields(corev1.Container{}, "TerminationMessagePath", "TerminationMessagePolicy", "ImagePullPolicy"),
}

func SyncObjectsToCluster(
		workspace *v1alpha1.Workspace,
		podAdditions []v1alpha1.PodAdditions,
		client runtimeClient.Client,
		scheme *runtime.Scheme) DeploymentProvisioningStatus {

	specDeployment, err := getSpecDeployment(workspace, podAdditions, scheme)
	if err != nil {
		return DeploymentProvisioningStatus{
			ProvisioningStatus: common.ProvisioningStatus{Err: err},
		}
	}

	clusterDeployment, err := getClusterDeployment(specDeployment.Name, workspace.Namespace, client)
	if err != nil {
		return DeploymentProvisioningStatus{
			ProvisioningStatus: common.ProvisioningStatus{Err: err},
		}
	}

	if clusterDeployment == nil {
		fmt.Printf("Creating deployment...\n")
		err := client.Create(context.TODO(), specDeployment)
		return DeploymentProvisioningStatus{
			ProvisioningStatus: common.ProvisioningStatus{
				Requeue: true,
				Err:     err,
			},
		}
	}

	if !cmp.Equal(specDeployment, clusterDeployment, deploymentDiffOpts) {
		fmt.Printf("Updating deployment...\n")
		fmt.Printf("\n\n%s\n\n", cmp.Diff(specDeployment, clusterDeployment, deploymentDiffOpts))
		//clusterDeployment.Spec = specDeployment.Spec
		err := client.Patch(context.TODO(), clusterDeployment, runtimeClient.MergeFrom(specDeployment))
		return DeploymentProvisioningStatus{
			ProvisioningStatus: common.ProvisioningStatus{Requeue: true, Err: err},
		}
	}

	deploymentReady := checkDeploymentStatus(clusterDeployment)
	if deploymentReady {
		return DeploymentProvisioningStatus{
			ProvisioningStatus: common.ProvisioningStatus{
				Continue: true,
			},
			Status: "Ready", // TODO
		}
	}

	return DeploymentProvisioningStatus{}
}

func checkDeploymentStatus(deployment *appsv1.Deployment) (ready bool) {
	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentAvailable && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func getSpecDeployment(workspace *v1alpha1.Workspace, podAdditionsList []v1alpha1.PodAdditions, scheme *runtime.Scheme) (*appsv1.Deployment, error) {
	replicas := int32(1)
	terminationGracePeriod := int64(1)
	rollingUpdateParam := intstr.FromInt(1)

	var user *int64
	if !config.ControllerCfg.IsOpenShift() {
		uID := int64(1234)
		user = &uID
	}

	podAdditions, err := mergePodAdditions(podAdditionsList)
	if err != nil {
		return nil, err
	}

	// TODO: Add che-rest-apis
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workspace.Status.WorkspaceId,
			Namespace: workspace.Namespace,
			Labels: map[string]string{
				model.WorkspaceIDLabel: workspace.Status.WorkspaceId,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": workspace.Status.WorkspaceId, // TODO
				},
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: "RollingUpdate",
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge:       &rollingUpdateParam,
					MaxUnavailable: &rollingUpdateParam,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      workspace.Status.WorkspaceId,
					Namespace: workspace.Namespace,
					Labels: map[string]string{
						"app": workspace.Status.WorkspaceId,
					},
				},
				Spec: corev1.PodSpec{
					InitContainers:                podAdditions.InitContainers,
					Containers:                    podAdditions.Containers,
					Volumes:                       append(podAdditions.Volumes, getPersistentVolumeClaim()),
					ImagePullSecrets:              podAdditions.PullSecrets,
					RestartPolicy:                 "Always",
					TerminationGracePeriodSeconds: &terminationGracePeriod,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser: user,
						FSGroup:   user,
					},
					ServiceAccountName:           "",
					AutomountServiceAccountToken: nil,
				},
			},
		},
	}

	err = controllerutil.SetControllerReference(workspace, deployment, scheme)
	if err != nil {
		return nil, err
	}

	return deployment, nil
}

func getClusterDeployment(name string, namespace string, client runtimeClient.Client) (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	err := client.Get(context.TODO(), namespacedName, deployment)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return deployment, nil
}

func mergePodAdditions(toMerge []v1alpha1.PodAdditions) (*v1alpha1.PodAdditions, error) {
	podAdditions := &v1alpha1.PodAdditions{}

	// "Set"s to store k8s object names and detect duplicates
	containerNames := map[string]bool{}
	initContainerNames := map[string]bool{}
	volumeNames := map[string]bool{}
	pullSecretNames := map[string]bool{}
	for _, additions := range toMerge {
		for annotKey, annotVal := range additions.Annotations {
			podAdditions.Annotations[annotKey] = annotVal
		}
		for labelKey, labelVal := range additions.Labels {
			podAdditions.Labels[labelKey] = labelVal
		}
		for _, container := range additions.Containers {
			if containerNames[container.Name] {
				return nil, fmt.Errorf("Duplicate containers in the workspace definition: %s", container.Name)
			}
			containerNames[container.Name] = true
			podAdditions.Containers = append(podAdditions.Containers, container)
		}

		for _, container := range additions.InitContainers {
			if initContainerNames[container.Name] {
				return nil, fmt.Errorf("Duplicate init containers in the workspace definition: %s", container.Name)
			}
			initContainerNames[container.Name] = true
			podAdditions.InitContainers = append(podAdditions.InitContainers, container)
		}

		for _, volume := range additions.Volumes {
			if volumeNames[volume.Name] {
				return nil, fmt.Errorf("Duplicate volumes in the workspace definition: %s", volume.Name)
			}
			volumeNames[volume.Name] = true
			podAdditions.Volumes = append(podAdditions.Volumes, volume)
		}

		for _, pullSecret := range additions.PullSecrets {
			if pullSecretNames[pullSecret.Name] {
				continue
			}
			pullSecretNames[pullSecret.Name] = true
			podAdditions.PullSecrets = append(podAdditions.PullSecrets, pullSecret)
		}
	}
	return podAdditions, nil
}

func getPersistentVolumeClaim() corev1.Volume {
	var workspaceClaim = corev1.PersistentVolumeClaimVolumeSource{
		ClaimName: config.ControllerCfg.GetWorkspacePVCName(),
	}
	pvcVolume := corev1.Volume{
		Name: config.ControllerCfg.GetWorkspacePVCName(),
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &workspaceClaim,
		},
	}
	return pvcVolume
}

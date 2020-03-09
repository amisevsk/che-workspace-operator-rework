package workspace

import (
	"errors"
	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	"github.com/che-incubator/che-workspace-operator/pkg/controller/workspace/config"
	"github.com/che-incubator/che-workspace-operator/pkg/controller/workspace/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *ReconcileWorkspace) createWorkspaceDeployment(workspace *v1alpha1.Workspace, podAdditionsList []v1alpha1.PodAdditions) (*appsv1.Deployment, error) {
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
		ObjectMeta: v1.ObjectMeta{
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
				ObjectMeta: v1.ObjectMeta{
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

	err = controllerutil.SetControllerReference(workspace, deployment, r.scheme)
	if err != nil {
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
				return nil, errors.New("Duplicate containers in the workspace definition: " + container.Name)
			}
			containerNames[container.Name] = true
			podAdditions.Containers = append(podAdditions.Containers, container)
		}

		for _, container := range additions.InitContainers {
			if initContainerNames[container.Name] {
				return nil, errors.New("Duplicate init conainers in the workspace definition: " + container.Name)
			}
			initContainerNames[container.Name] = true
			podAdditions.InitContainers = append(podAdditions.InitContainers, container)
		}

		for _, volume := range additions.Volumes {
			if volumeNames[volume.Name] {
				return nil, errors.New("Duplicate volumes in the workspace definition: " + volume.Name)
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

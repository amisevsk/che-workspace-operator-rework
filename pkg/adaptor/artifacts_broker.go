package adaptor

import (
	"encoding/json"
	"fmt"
	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	"github.com/che-incubator/che-workspace-operator/pkg/config"
	"github.com/eclipse/che-plugin-broker/model"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getArtifactsBrokerComponent(workspaceId, namespace string, components []v1alpha1.ComponentSpec) (*v1alpha1.ComponentDescription, *v1.ConfigMap, error) {
	const (
		configMapVolumeName = "broker-config-volume"
		configMapMountPath  = "/broker-config"
		configMapDataName   = "config.json"
	)
	configMapName := fmt.Sprintf("%s.broker-config-map", workspaceId)
	brokerImage := config.ControllerCfg.GetPluginArtifactsBrokerImage()
	brokerContainerName := "plugin-artifacts-broker"

	var fqns []model.PluginFQN
	for _, component := range components {
		fqns = append(fqns, getPluginFQN(component))
	}
	cmData, err := json.Marshal(fqns)
	if err != nil {
		return nil, nil, err
	}
	cm := &v1.ConfigMap{
		ObjectMeta: v12.ObjectMeta{
			Name:      configMapName,
			Namespace: namespace,
			Labels: map[string]string{
				config.WorkspaceIDLabel: workspaceId,
			},
		},
		Data: map[string]string{
			configMapDataName: string(cmData),
		},
	}

	cmMode := int32(0644)
	// Define volumes used by plugin broker
	cmVolume := v1.Volume{
		Name: configMapVolumeName,
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{
					Name: configMapName,
				},
				DefaultMode: &cmMode,
			},
		},
	}

	cmVolumeMounts := []v1.VolumeMount{
		{
			MountPath: configMapMountPath,
			Name:      configMapVolumeName,
			ReadOnly:  true,
		},
		{
			MountPath: config.PluginsMountPath,
			Name:      config.ControllerCfg.GetWorkspacePVCName(),
			SubPath:   workspaceId + "/plugins",
		},
	}

	initContainer := v1.Container{
		Name:                     brokerContainerName,
		Image:                    brokerImage,
		ImagePullPolicy:          v1.PullPolicy(config.ControllerCfg.GetSidecarPullPolicy()),
		VolumeMounts:             cmVolumeMounts,
		TerminationMessagePolicy: v1.TerminationMessageFallbackToLogsOnError,
		Args: []string{
			"--disable-push",
			"--runtime-id",
			fmt.Sprintf("%s:%s:%s", workspaceId, "default", "anonymous"),
			"--registry-address",
			config.ControllerCfg.GetPluginRegistry(),
			"--metas",
			fmt.Sprintf("%s/%s", configMapMountPath, configMapDataName),
		},
		Resources: v1.ResourceRequirements{
			Limits: v1.ResourceList{
				v1.ResourceMemory: resource.MustParse("150Mi"),
			},
			Requests: v1.ResourceList{
				v1.ResourceMemory: resource.MustParse("150Mi"),
			},
		},
	}

	brokerComponent := &v1alpha1.ComponentDescription{
		Name: "artifacts-broker",
		PodAdditions: v1alpha1.PodAdditions{
			InitContainers: []v1.Container{initContainer},
			Volumes:        []v1.Volume{cmVolume},
		},
	}

	return brokerComponent, cm, nil
}

func isArtifactsBrokerNecessary(metas []model.PluginMeta) bool {
	for _, meta := range metas {
		if len(meta.Spec.Extensions) > 0 {
			return true
		}
	}
	return false
}


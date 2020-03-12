package adaptor

import (
	"encoding/json"
	"fmt"
	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	"github.com/che-incubator/che-workspace-operator/pkg/common"
	"github.com/che-incubator/che-workspace-operator/pkg/config"
	metadataBroker "github.com/eclipse/che-plugin-broker/brokers/metadata"
	brokerModel "github.com/eclipse/che-plugin-broker/model"
	"github.com/eclipse/che-plugin-broker/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"strings"
)

func AdaptPluginComponents(workspaceId, namespace string, devfileComponents []v1alpha1.ComponentSpec) ([]v1alpha1.ComponentDescription, *corev1.ConfigMap, error) {
	var components []v1alpha1.ComponentDescription

	broker := metadataBroker.NewBroker(true)

	metas, aliases, err := getMetasForComponents(devfileComponents)
	if err != nil {
		return nil, nil, err
	}
	plugins, err := broker.ProcessPlugins(metas)
	if err != nil {
		return nil, nil, err
	}

	for _, plugin := range plugins {
		component, err := adaptChePluginToComponent(workspaceId, plugin)
		if err != nil {
			return nil, nil, err
		}
		if aliases[plugin.ID] != "" {
			component.Name = aliases[plugin.ID]
		}
		components = append(components, component)
	}

	var artifactsBrokerCM *corev1.ConfigMap
	if isArtifactsBrokerNecessary(metas) {
		artifactsBrokerComponent, configMap, err := getArtifactsBrokerComponent(workspaceId, namespace, devfileComponents)
		if err != nil {
			return nil, nil, err
		}
		components = append(components, *artifactsBrokerComponent)
		artifactsBrokerCM = configMap
	}

	return components, artifactsBrokerCM, nil
}

func getArtifactsBrokerComponent(workspaceId, namespace string, components []v1alpha1.ComponentSpec) (*v1alpha1.ComponentDescription, *corev1.ConfigMap, error) {
	const (
		configMapVolumeName = "broker-config-volume"
		configMapMountPath  = "/broker-config"
		configMapDataName   = "config.json"
	)
	configMapName := fmt.Sprintf("%s.broker-config-map", workspaceId)
	brokerImage := config.ControllerCfg.GetPluginArtifactsBrokerImage()
	brokerContainerName := "plugin-artifacts-broker"

	var fqns []brokerModel.PluginFQN
	for _, component := range components {
		fqns = append(fqns, getPluginFQN(component))
	}
	cmData, err := json.Marshal(fqns)
	if err != nil {
		return nil, nil, err
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
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
	cmVolume := corev1.Volume{
		Name: configMapVolumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: configMapName,
				},
				DefaultMode: &cmMode,
			},
		},
	}

	cmVolumeMounts := []corev1.VolumeMount{
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

	initContainer := corev1.Container{
		Name:                     brokerContainerName,
		Image:                    brokerImage,
		ImagePullPolicy:          corev1.PullPolicy(config.ControllerCfg.GetSidecarPullPolicy()),
		VolumeMounts:             cmVolumeMounts,
		TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
		Args: []string{
			"--disable-push",
			"--runtime-id",
			fmt.Sprintf("%s:%s:%s", workspaceId, "default", "anonymous"),
			"--registry-address",
			config.ControllerCfg.GetPluginRegistry(),
			"--metas",
			fmt.Sprintf("%s/%s", configMapMountPath, configMapDataName),
		},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("150Mi"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("150Mi"),
			},
		},
	}

	brokerComponent := &v1alpha1.ComponentDescription{
		Name: "artifacts-broker",
		PodAdditions: v1alpha1.PodAdditions{
			InitContainers: []corev1.Container{initContainer},
			Volumes:        []corev1.Volume{cmVolume},
		},
	}

	return brokerComponent, cm, nil
}

func adaptChePluginToComponent(workspaceId string, plugin brokerModel.ChePlugin) (v1alpha1.ComponentDescription, error) {
	var containers []corev1.Container
	containerDescriptions := map[string]v1alpha1.ContainerDescription{}
	for _, pluginContainer := range plugin.Containers {
		container, containerDescription, err := convertPluginContainer(workspaceId, plugin.ID, pluginContainer)
		if err != nil {
			return v1alpha1.ComponentDescription{}, err
		}
		containers = append(containers, container)
		containerDescriptions[container.Name] = containerDescription
	}
	var initContainers []corev1.Container
	for _, pluginInitContainer := range plugin.InitContainers {
		container, _, err := convertPluginContainer(workspaceId, plugin.ID, pluginInitContainer)
		if err != nil {
			return v1alpha1.ComponentDescription{}, err
		}
		initContainers = append(initContainers, container)
	}

	component := v1alpha1.ComponentDescription{
		Name: plugin.Name,
		PodAdditions: v1alpha1.PodAdditions{
			Containers:     containers,
			InitContainers: initContainers,
		},
		ComponentMetadata: v1alpha1.ComponentMetadata{
			Containers:                 containerDescriptions,
			ContributedRuntimeCommands: nil, // TODO
			Endpoints:                  createEndpointsFromPlugin(plugin),
		},
	}

	return component, nil
}

func createEndpointsFromPlugin(plugin brokerModel.ChePlugin) []v1alpha1.Endpoint {
	var endpoints []v1alpha1.Endpoint

	for _, pluginEndpoint := range plugin.Endpoints {
		attributes := map[v1alpha1.EndpointAttribute]string{}
		// Default value of http for protocol, may be overwritten by pluginEndpoint attributes
		attributes[v1alpha1.PROTOCOL_ENDPOINT_ATTRIBUTE] = "http"
		attributes[v1alpha1.PUBLIC_ENDPOINT_ATTRIBUTE] = strconv.FormatBool(pluginEndpoint.Public)
		for key, val := range pluginEndpoint.Attributes {
			attributes[v1alpha1.EndpointAttribute(key)] = val
		}
		endpoints = append(endpoints, v1alpha1.Endpoint{
			Name:       common.EndpointName(pluginEndpoint.Name),
			Port:       int64(pluginEndpoint.TargetPort),
			Attributes: attributes,
		})
	}

	return endpoints
}

func convertPluginContainer(workspaceId, pluginID string, brokerContainer brokerModel.Container) (corev1.Container, v1alpha1.ContainerDescription, error) {
	memorylimit := brokerContainer.MemoryLimit
	if memorylimit == "" {
		memorylimit = config.SidecarDefaultMemoryLimit
	}
	containerResources, err := adaptResourcesFromString(memorylimit)
	if err != nil {
		return corev1.Container{}, v1alpha1.ContainerDescription{}, err
	}

	var env []corev1.EnvVar
	for _, brokerEnv := range brokerContainer.Env {
		env = append(env, corev1.EnvVar{
			Name:  brokerEnv.Name,
			Value: brokerEnv.Value,
		})
	}

	var containerPorts []corev1.ContainerPort
	var portInts []int
	for _, brokerPort := range brokerContainer.Ports {
		containerPorts = append(containerPorts, corev1.ContainerPort{
			ContainerPort: int32(brokerPort.ExposedPort),
			Protocol:      corev1.ProtocolTCP,
		})
		portInts = append(portInts, brokerPort.ExposedPort)
	}

	container := corev1.Container{
		Name:            brokerContainer.Name,
		Image:           brokerContainer.Image,
		Command:         brokerContainer.Command,
		Args:            brokerContainer.Args,
		Ports:           containerPorts,
		Env:             env,
		Resources:       containerResources,
		VolumeMounts:    adaptVolumeMountsFromBroker(workspaceId, brokerContainer),
		ImagePullPolicy: corev1.PullAlways,
	}

	containerDescription := v1alpha1.ContainerDescription{
		Attributes: map[string]string{
			config.RestApisContainerSourceAttribute: config.RestApisRecipeSourceToolAttribute,
			config.RestApisPluginMachineAttribute:   pluginID,
		}, // TODO
		Ports: portInts,
	}

	return container, containerDescription, nil
}

func adaptVolumeMountsFromBroker(workspaceId string, brokerContainer brokerModel.Container) []corev1.VolumeMount {
	var volumeMounts []corev1.VolumeMount
	volumeName := config.ControllerCfg.GetWorkspacePVCName()

	// TODO: Handle ephemeral
	for _, brokerVolume := range brokerContainer.Volumes {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			SubPath:   fmt.Sprintf("%s/%s/", workspaceId, brokerVolume.Name),
			MountPath: brokerVolume.MountPath,
		})
	}
	if brokerContainer.MountSources {
		volumeMounts = append(volumeMounts, GetProjectSourcesVolumeMount(workspaceId))
	}

	return volumeMounts
}

func isArtifactsBrokerNecessary(metas []brokerModel.PluginMeta) bool {
	for _, meta := range metas {
		if len(meta.Spec.Extensions) > 0 {
			return true
		}
	}
	return false
}

func getMetasForComponents(components []v1alpha1.ComponentSpec) (metas []brokerModel.PluginMeta, aliases map[string]string, err error) {
	defaultRegistry := config.ControllerCfg.GetPluginRegistry()
	ioUtils := utils.New()
	aliases = map[string]string{}
	for _, component := range components {
		fqn := getPluginFQN(component)
		meta, err := utils.GetPluginMeta(fqn, defaultRegistry, ioUtils)
		if err != nil {
			return nil, nil, err
		}
		metas = append(metas, *meta)
		aliases[meta.ID] = component.Alias
	}
	err = utils.ResolveRelativeExtensionPaths(metas, defaultRegistry)
	if err != nil {
		return nil, nil, err
	}
	return metas, aliases, nil
}

func getPluginFQN(component v1alpha1.ComponentSpec) brokerModel.PluginFQN {
	var pluginFQN brokerModel.PluginFQN
	registryAndID := strings.Split(component.Id, "#")
	if len(registryAndID) == 2 {
		pluginFQN.Registry = registryAndID[0]
		pluginFQN.ID = registryAndID[1]
	} else if len(registryAndID) == 1 {
		pluginFQN.ID = registryAndID[0]
	}
	pluginFQN.Reference = component.Reference
	return pluginFQN
}

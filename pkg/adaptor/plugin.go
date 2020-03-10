package adaptor

import (
	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	"github.com/che-incubator/che-workspace-operator/pkg/controller/workspace/config"
	"github.com/che-incubator/che-workspace-operator/pkg/controller/workspace/model"
	"github.com/che-incubator/che-workspace-operator/pkg/controller/workspace/server"
	metadataBroker "github.com/eclipse/che-plugin-broker/brokers/metadata"
	brokerModel "github.com/eclipse/che-plugin-broker/model"
	"github.com/eclipse/che-plugin-broker/utils"
	corev1 "k8s.io/api/core/v1"
	"strconv"
	"strings"
)

func AdaptPluginComponents(devfileComponents []v1alpha1.ComponentSpec) ([]v1alpha1.ComponentDescription, error) {
	var components []v1alpha1.ComponentDescription

	broker := metadataBroker.NewBroker(true)

	metas, err := getMetasForComponents(devfileComponents)
	if err != nil {
		return nil, err
	}
	plugins, err := broker.ProcessPlugins(metas)
	if err != nil {
		return nil, err
	}

	for _, plugin := range plugins {
		component, err := adaptChePluginToComponent(plugin)
		if err != nil {
			return nil, err
		}
		components = append(components, component)
	}

	return components, nil
}

func adaptChePluginToComponent(plugin brokerModel.ChePlugin) (v1alpha1.ComponentDescription, error) {
	component := v1alpha1.ComponentDescription{}

	var containers []corev1.Container
	for _, pluginContainer := range plugin.Containers {
		container, containerDescription, err := convertPluginContainer(pluginContainer, plugin.ID)
		if err != nil {
			return component, err
		}
		containers = append(containers, container)
		component.ComponentMetadata = v1alpha1.ComponentMetadata{
			Containers: map[string]v1alpha1.ContainerDescription{
				container.Name: containerDescription,
			},
			ContributedRuntimeCommands: nil, // TODO Handle this where it makes sense
			Endpoints:                  createEndpointsFromPlugin(plugin),
		}
		// TODO: Use aliases to set names?
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
			Name:       pluginEndpoint.Name,
			Port:       int64(pluginEndpoint.TargetPort),
			Attributes: attributes,
		})
	}

	return endpoints
}

func convertPluginContainer(brokerContainer brokerModel.Container, pluginID string) (corev1.Container, v1alpha1.ContainerDescription, error) {
	memorylimit := brokerContainer.MemoryLimit
	if memorylimit == "" {
		memorylimit = "128Mi"
	} // todo
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
			Protocol:      "TCP",
		})
		portInts = append(portInts, int(brokerPort.ExposedPort))
	}

	container := corev1.Container{
		Name:            brokerContainer.Name,
		Image:           brokerContainer.Image,
		Command:         brokerContainer.Command,
		Args:            brokerContainer.Args,
		Ports:           containerPorts,
		Env:             env,
		Resources:       containerResources,
		VolumeMounts:    adaptVolumeMountsFromBroker(brokerContainer),
		ImagePullPolicy: corev1.PullAlways,
	}

	containerDescription := v1alpha1.ContainerDescription{
		Attributes: map[string]string{
			server.CONTAINER_SOURCE_ATTRIBUTE: server.TOOL_CONTAINER_SOURCE,
			server.PLUGIN_MACHINE_ATTRIBUTE:   pluginID,
		}, // TODO
		Ports: portInts,
	}

	return container, containerDescription, nil
}

func adaptVolumeMountsFromBroker(brokerContainer brokerModel.Container) []corev1.VolumeMount {
	var volumeMounts []corev1.VolumeMount

	// TODO: Handle ephemeral
	for _, devfileVolume := range brokerContainer.Volumes {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      devfileVolume.Name,
			MountPath: devfileVolume.MountPath,
		})
	}
	if brokerContainer.MountSources {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			MountPath: model.DefaultProjectsSourcesRoot,
			Name:      model.DefaultPluginsVolumeName,
		})
	}

	return volumeMounts
}

func getMetasForComponents(components []v1alpha1.ComponentSpec) ([]brokerModel.PluginMeta, error) {
	defaultRegistry := config.ControllerCfg.GetPluginRegistry()
	ioUtils := utils.New()
	var metas []brokerModel.PluginMeta
	for _, component := range components {
		fqn := getPluginFQN(component)
		meta, err := utils.GetPluginMeta(fqn, defaultRegistry, ioUtils)
		if err != nil {
			return nil, err
		}
		metas = append(metas, *meta)
	}
	utils.ResolveRelativeExtensionPaths(metas, defaultRegistry)
	return metas, nil
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

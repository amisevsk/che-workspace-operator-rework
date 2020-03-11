package adaptor

import (
	"fmt"
	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	"github.com/che-incubator/che-workspace-operator/pkg/config"
	corev1 "k8s.io/api/core/v1"
	"strings"
)

func AdaptDockerimageComponents(workspaceId string, devfileComponents []v1alpha1.ComponentSpec) ([]v1alpha1.ComponentDescription, error) {
	var components []v1alpha1.ComponentDescription
	for _, devfileComponent := range devfileComponents {
		if devfileComponent.Type != v1alpha1.Dockerimage {
			return nil, fmt.Errorf("cannot adapt non-dockerfile type component %s in docker adaptor", devfileComponent.Alias)
		}
		component, err := adaptDockerimageComponent(workspaceId, devfileComponent)
		if err != nil {
			return nil, err
		}
		components = append(components, component)
	}

	return components, nil
}

func adaptDockerimageComponent(workspaceId string, devfileComponent v1alpha1.ComponentSpec) (v1alpha1.ComponentDescription, error) {
	component := v1alpha1.ComponentDescription{}

	container, containerDescription, err := getContainerFromDevfile(devfileComponent)
	if devfileComponent.MountSources {
		container.VolumeMounts = append(container.VolumeMounts, GetProjectSourcesVolumeMount(workspaceId))
	}
	if err != nil {
		return component, nil
	}
	component.PodAdditions.Containers = []corev1.Container{container}

	component.PodAdditions.Volumes = adaptVolumesFromDevfile(devfileComponent.Volumes)

	component.ComponentMetadata = v1alpha1.ComponentMetadata{
		Containers: map[string]v1alpha1.ContainerDescription{
			container.Name: containerDescription,
		},
		ContributedRuntimeCommands: nil, // TODO Handle this where it makes sense
		Endpoints:                  devfileComponent.Endpoints,
	}

	return component, nil
}

func getContainerFromDevfile(devfileComponent v1alpha1.ComponentSpec) (corev1.Container, v1alpha1.ContainerDescription, error) {
	containerResources, err := adaptResourcesFromString(devfileComponent.MemoryLimit)
	if err != nil {
		return corev1.Container{}, v1alpha1.ContainerDescription{}, err
	}
	containerEndpoints, endpointInts := endpointsToContainerPorts(devfileComponent.Endpoints)

	var env []corev1.EnvVar
	for _, devfileEnvVar := range devfileComponent.Env {
		env = append(env, corev1.EnvVar{
			Name:  devfileEnvVar.Name,
			Value: strings.ReplaceAll(devfileEnvVar.Value, "$(CHE_PROJECTS_ROOT)", config.DefaultProjectsSourcesRoot),
		})
	}
	env = append(env, corev1.EnvVar{
		Name:  "CHE_MACHINE_NAME",
		Value: devfileComponent.Alias,
	})

	container := corev1.Container{
		Name:         devfileComponent.Alias,
		Image:        devfileComponent.Image,
		Command:      devfileComponent.Command,
		Args:         devfileComponent.Args,
		Ports:        containerEndpoints,
		Env:          env,
		Resources:    containerResources,
		VolumeMounts: adaptVolumesMountsFromDevfile(devfileComponent.Volumes),
		ImagePullPolicy: corev1.PullAlways,
	}

	containerDescription := v1alpha1.ContainerDescription{
		Attributes: map[string]string{
			config.RestApisContainerSourceAttribute: config.RestApisContainerSourceAttribute,
		},
		Ports:      endpointInts,
	}
	return container, containerDescription, nil
}

func endpointsToContainerPorts(endpoints []v1alpha1.Endpoint) ([]corev1.ContainerPort, []int) {
	var containerPorts []corev1.ContainerPort
	var containerEndpoints []int

	for _, endpoint := range endpoints {
		containerPorts = append(containerPorts, corev1.ContainerPort{
			Name:          endpoint.Name,
			ContainerPort: int32(endpoint.Port),
			Protocol:      corev1.Protocol(endpoint.Attributes[v1alpha1.PROTOCOL_ENDPOINT_ATTRIBUTE]),
		})
		containerEndpoints = append(containerEndpoints, int(endpoint.Port))
	}

	return containerPorts, containerEndpoints
}

func adaptVolumesMountsFromDevfile(devfileVolumes []v1alpha1.Volume) []corev1.VolumeMount {
	var volumeMounts []corev1.VolumeMount

	for _, devfileVolume := range devfileVolumes {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      devfileVolume.Name,
			MountPath: devfileVolume.ContainerPath,
		})
	}
	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		MountPath: config.DefaultProjectsSourcesRoot,
		Name:      config.DefaultPluginsVolumeName,
	})

	return volumeMounts
}

func adaptVolumesFromDevfile(devfileVolumes []v1alpha1.Volume) []corev1.Volume {
	var volumes []corev1.Volume

	for _, devfileVolume := range devfileVolumes {
		volumes = append(volumes, corev1.Volume{
			Name: devfileVolume.Name,
			VolumeSource: corev1.VolumeSource{
				// TODO: temp workaround
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	}

	volumes = append(volumes, corev1.Volume{
		Name: config.DefaultPluginsVolumeName,
		VolumeSource: corev1.VolumeSource{
			// TODO: temp workaround
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	return volumes
}

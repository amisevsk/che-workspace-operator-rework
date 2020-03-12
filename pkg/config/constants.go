//
// Copyright (c) 2019-2020 Red Hat, Inc.
// This program and the accompanying materials are made
// available under the terms of the Eclipse Public License 2.0
// which is available at https://www.eclipse.org/legal/epl-2.0/
//
// SPDX-License-Identifier: EPL-2.0
//
// Contributors:
//   Red Hat, Inc. - initial API and implementation
//

package config

// Internal constants
const (
	//default URL for accessing Che Rest API Emulator from Workspace containers
	DefaultApiEndpoint = "http://localhost:9999/api/"

	DefaultProjectsSourcesRoot = "/projects"

	DefaultPluginsVolumeName = "plugins"
	PluginsMountPath = "/plugins"

	CheOriginalName = "workspace"

	AuthEnabled = "false"

	ServiceAccount = "che-workspace"

	SidecarDefaultMemoryLimit = "128M"
	PVCStorageSize            = "1Gi"

	//RuntimeAdditionalInfo is a key of workspaceStatus.AdditionalInfo where runtime info is stored
	RuntimeAdditionalInfo = "org.eclipse.che.workspace/runtime"

	//RuntimeAdditionalInfo is a key of workspaceStatus.AdditionalInfo info where component statuses info is stored
	ComponentStatusesAdditionalInfo = "org.eclipse.che.workspace/componentstatuses"

	//WorkspaceIDLabel is label key to store workspace identifier
	WorkspaceIDLabel = "che.workspace_id"

	//WorkspaceNameLabel is label key to store workspace identifier
	WorkspaceNameLabel = "che.workspace_name"

	//CheOriginalNameLabel is label key to original name
	CheOriginalNameLabel = "che.original_name"

	WorkspaceCreatorAnnotation = "org.eclipse.che.workspace/creator"
)

// Constants for che-rest-apis
const(
	// Attribute of Runtime Machine to mark source of the container.
	RestApisContainerSourceAttribute = "source"
	RestApisPluginMachineAttribute = "plugin"

	// Mark containers applied to workspace with help recipe definition.
	RestApisRecipeSourceContainerAttribute = "recipe"

	// Mark containers created workspace api like tooling for user development.
	RestApisRecipeSourceToolAttribute = "tool"
)
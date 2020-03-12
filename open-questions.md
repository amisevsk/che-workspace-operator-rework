## Todo
- Move runtime information to a configmap and mount it to che-rest-apis, avoiding need to inspect cluster
- Do same for devfile
- Sometimes we update or patch an out-of-date object, logging an error; this should be handled
- Do we still need to manage env var substitution in workspace commands on the controller side? c.f. interpolate in mainline repo

## Design questions
1. How should mountSources be handled? 
    1. VolumeMounts *and* Volumes are defined in adaptors (i.e. dockerfile adaptor includes `/projects` in its status)
        - Means we have to deduplicate volumes when merging for main deployment, since multiple components could contribute it
        - Have to sync with controller's settings to get PVC name, etc.
    1. VolumeMount defined in adaptor, Volume defined in main controller
        - Have to sync between controllers, as half of the mount is defined in workspace controller and other half in component controller (e.g. volume and mount name have to match)
        - Outputs of component controller is not "complete" -- workspace controller has to remember to add
    1. VolumeMounts *and* Volumes are defined in workspace controller
        - Config is synced, and no need to de-dupe volumemounts
        - MountSources state is lost by default; have to pass back in ComponentMetadata, etc.
            - Complicates matching containers in podadditions to component
        - Hard to match which plugin containers get sources and which don't
            
    In current main repo: option 2 
        - converter.go: setupPersistentVolumeClaim adds volume to "main deployment"
        - each devfile component handler mounts sources if defined in devfile.
    Current approach: option 2

1. What does the `discoverable` annotation on an Endpoint mean? 
    - Che docs:
        > discoverable: If an endpoint is discoverable, it means that it can be accessed using its name as the hostname within the workspace containers (in the Kubernetes parlance, a service is created for it with the provided name).
     
    - For dockerimage components, it means that no service is created but port is open on container, which is a bit strange, since we can always bypass the 
    - No idea how it's supposed to work for plugins, since if we respect the docs for cloud-shell, we get a nonfunctioning workspace (che machine exec endpoint is not discoverable)    
    
1. CheWorkspaceCommand appears to be incompatible with Devfile CommandSpec
    - Devfile command defines actions as an array, CheWorkspaceCommand matches name to a single action
    
1. Alias vs Name vs Container name
    - 
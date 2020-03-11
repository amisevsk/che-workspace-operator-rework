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

# Che workspace operator

This repo is intended to be a proof-of-concept/proposal for a redesign of the [Che workspace operator](https://github.com/che-incubator/che-workspace-operator). The main goals are

1. Improve the main reconcile loop to delegate time-consuming steps to subcontrollers
2. Establish a clearer separation of concerns between different components of the controller, and simplify how data flows between different steps in the reconcile loop
3. Allow workspace routing to contribute to the main workspace deployment, in order to support in-deployment proxy containers for auth
4. Do full reconciles on all elements of a workspace; if a route is deleted or modified it should be recreated.

The current design doc is in the [docs](docs/design.adoc) directory.

## Testing

The makefile from the main [Che workspace operator](https://github.com/che-incubator/che-workspace-operator) is ported here. As a result, deploying should be as simple as `make deploy`. The same environment variables are supported.

One difference currently is that the dockerfile *does not* include a build step, so the project must be built using operator-sdk instead of `make docker`:

```
operator-sdk build <image:tag> \
  --verbose \
  --go-build-args "-o build/_output/bin/che-workspace-operator"
```
where the build-args are required to make the binary name match what is expected (otherwise it derives from the folder name).

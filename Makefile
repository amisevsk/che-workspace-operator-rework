IMG ?= quay.io/che-incubator/che-workspace-controller:7.1.0
NAMESPACE ?= che-workspace-controller
TOOL ?= oc
CLUSTER_IP ?= 192.168.99.100
PULL_POLICY ?= Always
WEBHOOK_ENABLED ?= false

all: help

_set_context:
	$(TOOL) create namespace $(NAMESPACE) || true
ifeq ($(TOOL),oc)
	$(TOOL) project $(NAMESPACE)
else
	$(TOOL) config set-context $($(TOOL) config current-context) --namespace=$(NAMESPACE)
endif

_deploy_registry:
	$(TOOL) apply -f ./deploy/registry/local
ifeq ($(TOOL),oc)
	$(TOOL) apply -f ./deploy/registry/local/os
else
	sed -i "s|192.168.99.100|$(CLUSTER_IP)|g" ./deploy/registry/local/k8s/ingress.yaml
	$(TOOL) apply -f ./deploy/registry/local/k8s
endif

_set_registry_url:
ifeq ($(TOOL),oc)
	$(eval PLUGIN_REGISTRY_HOST := $(shell $(TOOL) get route che-plugin-registry -n $(NAMESPACE) -o jsonpath='{.spec.host}' || echo ""))
else
	$(eval PLUGIN_REGISTRY_HOST := $(shell $(TOOL) get ingress che-plugin-registry -n $(NAMESPACE) -o jsonpath='{.spec.rules[0].host}' || echo ""))
endif

_update_yamls: _set_registry_url
	sed -i "s|plugin.registry.url: .*|plugin.registry.url: http://$(PLUGIN_REGISTRY_HOST)|g" ./deploy/controller_config.yaml
	sed -i "s|image: .*|image: $(IMG)|g" ./deploy/controller.yaml
	sed -i "s|imagePullPolicy: Always|imagePullPolicy: $(PULL_POLICY)|g" ./deploy/controller.yaml
	sed -i "s|che.webhooks.enabled: .*|che.webhooks.enabled: $(WEBHOOK_ENABLED)|g" ./deploy/controller.yaml

_reset_yamls: _set_registry_url
	sed -i "s|http://$(PLUGIN_REGISTRY_HOST)|http://che-plugin-registry.192.168.99.100.nip.io/v3|g" ./deploy/controller_config.yaml
	sed -i "s|image: $(IMG)|image: quay.io/che-incubator/che-workspace-controller:nightly|g" ./deploy/controller.yaml
	sed -i "s|imagePullPolicy: $(PULL_POLICY)|imagePullPolicy: Always|g" ./deploy/controller.yaml
	sed -i "s|che.webhooks.enabled: .*|che.webhooks.enabled: "false"|g" ./deploy/controller.yaml

_update_crds:
	$(TOOL) apply -f ./deploy/crds
	$(TOOL) apply -f ./deploy/controller_config.yaml

_deploy_controller:
	$(TOOL) apply -f ./deploy

### docker: build and push docker image
docker:
	docker build -t $(IMG) -f ./build/Dockerfile .
	docker push $(IMG)

### webhook: generate certificates for webhooks and deploy to cluster
webhook:
ifeq ($(WEBHOOK_ENABLED),true)
	./deploy/webhook-server-certs/deploy-webhook-server-certs.sh oc
else
	echo "Webhooks disabled, skipping certificate generation"
endif

### deploy: deploy controller to cluster
deploy: _set_context _deploy_registry _update_yamls _update_crds webhook _deploy_controller _reset_yamls

### restart: restart cluster controller deployment
restart:
ifeq ($(TOOL),oc)
	oc patch deployment/che-workspace-controller \
		-n che-workspace-controller \
		--patch "{\"spec\":{\"template\":{\"metadata\":{\"annotations\":{\"kubectl.kubernetes.io/restartedAt\":\"$(date --iso-8601=seconds)\"}}}}}"
else
	kubectl rollout restart -n $(NAMESPACE) che-workspace-controller
endif

### rollout: rebuild and push docker image and restart cluster deployment
rollout: docker restart

### update: update CRDs defined on cluster
update: _update_yamls _update_crds _reset_yamls

### uninstall: remove namespace and all CRDs from cluster
uninstall:
	$(TOOL) delete namespace $(NAMESPACE)
	$(TOOL) delete customresourcedefinitions.apiextensions.k8s.io workspaceroutings.workspace.che.eclipse.org
	$(TOOL) delete customresourcedefinitions.apiextensions.k8s.io workspaces.workspace.che.eclipse.org

### local: set up cluster for local development
local: _set_context _deploy_registry _set_registry_url _update_yamls _update_crds _reset_yamls

### fmt: format all go files in repository
fmt:
	go fmt -x ./...

.PHONY: help
### help: print this message
help: Makefile
	@echo "Available rules:"
	@sed -n 's/^### /    /p' $< | column -t -s ':' -o "       -"
	@echo ""
	@echo "Supported environment variables:"
	@echo "    IMG             - Image used for controller"
	@echo "    NAMESPACE       - Namespace to use for deploying controller"
	@echo "    TOOL            - CLI tool for interfacing with the cluster: kubectl or oc; if oc is used, deployment is tailored to OpenShift, otherwise Kubernetes"
	@echo "    CLUSTER_IP      - For Kubernetes only, the ip address of the cluster (minikube ip)"
	@echo "    PULL_POLICY     - Image pull policy for controller"
	@echo "    WEBHOOK_ENABLED - Whether webhooks should be enabled in the deployment"

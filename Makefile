IMG ?= quay.io/che-incubator/che-workspace-controller:7.1.0
NAMESPACE ?= che-workspace-controller
TOOL ?= oc
CLUSTER_IP ?= 192.168.99.100
PULL_POLICY ?= Always

all: deploy

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

_reset_yamls: _set_registry_url
	sed -i "s|http://$(PLUGIN_REGISTRY_HOST)|http://che-plugin-registry.192.168.99.100.nip.io/v3|g" ./deploy/controller_config.yaml
	sed -i "s|image: $(IMG)|image: quay.io/che-incubator/che-workspace-controller:nightly|g" ./deploy/controller.yaml
	sed -i "s|imagePullPolicy: $(PULL_POLICY)|imagePullPolicy: Always|g" ./deploy/controller.yaml

_update_crds:
	$(TOOL) apply -f ./deploy/crds
	$(TOOL) apply -f ./deploy/controller_config.yaml

_deploy_controller:
	$(TOOL) apply -f ./deploy

docker:
	docker build -t $(IMG) -f ./build/Dockerfile .

webhook:
	./deploy/webhook-server-certs/deploy-webhook-server-certs.sh oc

deploy: _set_context _deploy_registry _update_yamls _update_crds _deploy_controller _reset_yamls

local: _set_context _deploy_registry _set_registry_url _update_yamls _udpate_crds _reset_yamls

fmt:
	go fmt -x ./...

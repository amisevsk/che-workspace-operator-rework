package solvers

import (
	"fmt"
	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	"github.com/che-incubator/che-workspace-operator/pkg/common"
	routeV1 "github.com/openshift/api/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strconv"
)

type OpenShiftOAuthSolver struct{}

var _ RoutingSolver = (*OpenShiftOAuthSolver)(nil)

type proxyEndpoint struct {
	machineName            string
	upstreamEndpoint       v1alpha1.Endpoint
	publicEndpoint         v1alpha1.Endpoint
	publicEndpointHttpPort int64
}

func (s *OpenShiftOAuthSolver) GetSpecObjects(spec v1alpha1.WorkspaceRoutingSpec, workspaceMeta WorkspaceMetadata) RoutingObjects {
	var exposedEndpoints = map[string][]v1alpha1.ExposedEndpoint{}
	proxy, noProxy := getProxiedEndpoints(spec)
	defaultIngresses, defaultEndpoints := getIngressesForSpec(noProxy, workspaceMeta)
	for machineName, machineEndpoints := range defaultEndpoints {
		exposedEndpoints[machineName] = append(exposedEndpoints[machineName], machineEndpoints...)
	}

	portMappings := getProxyEndpointMappings(proxy, workspaceMeta)
	var proxyPorts = map[string][]v1alpha1.Endpoint{}
	for _, proxyEndpoint := range portMappings {
		proxyPorts[proxyEndpoint.machineName] = append(proxyPorts[proxyEndpoint.machineName], proxyEndpoint.publicEndpoint)
	}
	for machineName, machineEndpoints := range noProxy {
		proxyPorts[machineName] = append(proxyPorts[machineName], machineEndpoints...)
	}
	// Use common service for all unproxied endpoints
	proxyServices := getServicesForEndpoints(proxyPorts, workspaceMeta)
	for idx := range proxyServices {
		proxyServices[idx].Annotations = map[string]string{
			"service.alpha.openshift.io/serving-cert-secret-name": "proxy-tls", // TODO: Find a better way to do this
		}
	}

	routes, proxyEndpoints, podAdditions := s.getProxyRoutes(proxy, workspaceMeta, portMappings)
	for machineName, machineEndpoints := range proxyEndpoints {
		exposedEndpoints[machineName] = append(exposedEndpoints[machineName], machineEndpoints...)
	}

	return RoutingObjects{
		Services:         proxyServices,
		Ingresses:        defaultIngresses,
		Routes:           routes,
		PodAdditions:     podAdditions,
		ExposedEndpoints: exposedEndpoints,
	}
}

func (s *OpenShiftOAuthSolver) getProxyRoutes(
		endpoints map[string][]v1alpha1.Endpoint,
		workspaceMeta WorkspaceMetadata,
		portMappings map[string]proxyEndpoint) ([]routeV1.Route, map[string][]v1alpha1.ExposedEndpoint, *v1alpha1.PodAdditions) {

	var routes []routeV1.Route
	var exposedEndpoints = map[string][]v1alpha1.ExposedEndpoint{}
	var podAdditions *v1alpha1.PodAdditions

	for machineName, machineEndpoints := range endpoints {
		for _, upstreamEndpoint := range machineEndpoints {
			proxyEndpoint := portMappings[upstreamEndpoint.Name]
			endpoint := proxyEndpoint.publicEndpoint
			// TODO: Check if there is a limit on using endpoint.Name here as there is for ingresses
			targetEndpoint := intstr.FromInt(int(endpoint.Port))
			endpointName := common.EndpointName(endpoint.Name)
			hostname := fmt.Sprintf("%s-%s", endpointName, workspaceMeta.IngressGlobalDomain)

			// TODO: Currently, each proxied endpoint requires its own container and cookie; as such,
			// we cannot proxy the terminal route.
			var tls *routeV1.TLSConfig = nil
			if endpoint.Attributes[v1alpha1.SECURE_ENDPOINT_ATTRIBUTE] == "true" {
				if endpoint.Attributes[v1alpha1.TYPE_ENDPOINT_ATTRIBUTE] == "terminal" {
					tls = &routeV1.TLSConfig{
						Termination:                   routeV1.TLSTerminationEdge,
						InsecureEdgeTerminationPolicy: routeV1.InsecureEdgeTerminationPolicyRedirect,
					}
				} else {
					tls = &routeV1.TLSConfig{
						Termination:                   routeV1.TLSTerminationReencrypt,
						InsecureEdgeTerminationPolicy: routeV1.InsecureEdgeTerminationPolicyRedirect,
					}
				}
			}
			routes = append(routes, routeV1.Route{
				ObjectMeta: metav1.ObjectMeta{
					Name:      endpointName,
					Namespace: workspaceMeta.Namespace,
					Labels: map[string]string{
						"app": workspaceMeta.WorkspaceId + "oauth", // TODO: Ingresses get routes by default on OpenShift
					},
				},
				Spec: routeV1.RouteSpec{
					Host: hostname,
					To: routeV1.RouteTargetReference{
						Kind: "Service",
						Name: "service-" + workspaceMeta.WorkspaceId, // TODO: Copied from service func above
					},
					Port: &routeV1.RoutePort{
						TargetPort: targetEndpoint,
					},
					TLS: tls,
				},
			})
			exposedEndpoints[machineName] = append(exposedEndpoints[machineName], v1alpha1.ExposedEndpoint{
				Name:       endpoint.Name,
				Url:        hostname,
				Attributes: endpoint.Attributes,
			})
		}
	}
	podAdditions = getProxyPodAdditions(portMappings, workspaceMeta)
	return routes, exposedEndpoints, podAdditions
}



func getProxiedEndpoints(spec v1alpha1.WorkspaceRoutingSpec) (proxy, noProxy map[string][]v1alpha1.Endpoint) {
	proxy = map[string][]v1alpha1.Endpoint{}
	noProxy = map[string][]v1alpha1.Endpoint{}
	for machineName, machineEndpoints := range spec.Endpoints {
		for _, endpoint := range machineEndpoints {
			// TODO: Meaning of v1alpha1.PUBLIC_ENDPOINT_ATTRIBUTE = true is unclear
			if endpointNeedsProxy(endpoint) {
				proxy[machineName] = append(proxy[machineName], endpoint)
			} else {
				noProxy[machineName] = append(noProxy[machineName], endpoint)
			}
		}
	}
	return
}

func getProxyEndpointMappings(
		endpoints map[string][]v1alpha1.Endpoint, workspaceMeta WorkspaceMetadata) map[string]proxyEndpoint {
	proxyHttpsPort := 4400
	proxyHttpPort := int64(4180)

	var proxyEndpoints = map[string]proxyEndpoint{}
	for machineName, machineEndpoints := range endpoints {
		for _, endpoint := range machineEndpoints {
			proxyEndpoints[endpoint.Name] = proxyEndpoint{
				machineName:      machineName,
				upstreamEndpoint: endpoint,
				publicEndpoint: v1alpha1.Endpoint{
					// TODO: Ensure copying over attributes is correct
					Attributes: endpoint.Attributes,
					Name:       fmt.Sprintf("%s-proxy-%s", workspaceMeta.WorkspaceId, strconv.Itoa(proxyHttpsPort)),
					Port:       int64(proxyHttpsPort),
				},
				publicEndpointHttpPort: proxyHttpPort,
			}
			proxyHttpsPort++
			proxyHttpPort++
		}
	}

	return proxyEndpoints
}

func endpointNeedsProxy(endpoint v1alpha1.Endpoint) bool {
	publicAttr, exists := endpoint.Attributes[v1alpha1.PUBLIC_ENDPOINT_ATTRIBUTE]
	endpointIsPublic := !exists || (publicAttr == "true")
	return endpointIsPublic &&
			endpoint.Attributes[v1alpha1.SECURE_ENDPOINT_ATTRIBUTE] == "true" &&
			endpoint.Attributes[v1alpha1.TYPE_ENDPOINT_ATTRIBUTE] != "terminal"
}

package solvers

import (
	"fmt"
	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	"github.com/che-incubator/che-workspace-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strconv"
)

type WorkspaceMetadata struct {
	WorkspaceId     string
	Namespace       string
	PodSelector     map[string]string
	IngressGlobalDomain string
}

func getServicesForEndpoints(endpoints map[string][]v1alpha1.Endpoint, workspaceMeta WorkspaceMetadata) []corev1.Service {
	var servicePorts []corev1.ServicePort
	for _, machineEndpoints := range endpoints {
		for _, endpoint := range machineEndpoints {

			if endpoint.Attributes[v1alpha1.DISCOVERABLE_ATTRIBUTE] != "true" {
				//continue // TODO: Unclear how this is supposed to work?
			}
			servicePorts = append(servicePorts, corev1.ServicePort{
				Name:       common.EndpointName(endpoint.Name),
				Protocol:   corev1.ProtocolTCP,
				Port:       int32(endpoint.Port),
				TargetPort: intstr.FromInt(int(endpoint.Port)),
			})
		}
	}

	return []corev1.Service{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "service-" + workspaceMeta.WorkspaceId, // TODO?
				Namespace: workspaceMeta.Namespace,
				Labels: map[string]string{
					"app": workspaceMeta.WorkspaceId,
				},
			},
			Spec: corev1.ServiceSpec{
				Ports:    servicePorts,
				Selector: workspaceMeta.PodSelector,
				Type:     corev1.ServiceTypeClusterIP,
			},
		},
	}
}

func getIngressesForSpec(endpoints map[string][]v1alpha1.Endpoint, workspaceMeta WorkspaceMetadata) ([]v1beta1.Ingress, map[string][]v1alpha1.ExposedEndpoint) {
	var ingresses []v1beta1.Ingress
	exposedEndpoints := map[string][]v1alpha1.ExposedEndpoint{}

	for machineName, machineEndpoints := range endpoints {
		for _, endpoint := range machineEndpoints {
			if endpoint.Attributes[v1alpha1.PUBLIC_ENDPOINT_ATTRIBUTE] != "true" {
				//continue // TODO: Unclear how this is supposed to work?
			}
			// Note: there is an additional limitation on target endpoint here: must be a DNS name fewer than 15 chars long
			// In general, endpoint.Name _cannot_ be used here
			var targetEndpoint intstr.IntOrString
			targetEndpoint = intstr.FromInt(int(endpoint.Port))

			endpointName := common.EndpointName(endpoint.Name)
			ingressHostname := fmt.Sprintf("%s-%s-%s.%s",
				workspaceMeta.WorkspaceId, endpointName, strconv.FormatInt(endpoint.Port, 10), workspaceMeta.IngressGlobalDomain)
			ingresses = append(ingresses, v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-%s", workspaceMeta.WorkspaceId, endpointName),
					Namespace: workspaceMeta.Namespace,
					Labels: map[string]string{
						"app": workspaceMeta.WorkspaceId,
					},
					Annotations: ingressAnnotations,
				},
				Spec: v1beta1.IngressSpec{
					Rules: []v1beta1.IngressRule{
						{
							Host: ingressHostname,
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{
										{
											Backend: v1beta1.IngressBackend{
												ServiceName: "service-" + workspaceMeta.WorkspaceId, // TODO: Copied from service func above
												ServicePort: targetEndpoint,
											},
										},
									},
								},
							},
						},
					},
				},
			})
			exposedEndpoints[machineName] = append(exposedEndpoints[machineName], v1alpha1.ExposedEndpoint{
				Name:       endpoint.Name,
				Url:        ingressHostname,
				Attributes: endpoint.Attributes,
			})
		}
	}
	return ingresses, exposedEndpoints
}

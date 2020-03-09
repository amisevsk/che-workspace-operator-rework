package workspacerouting

import (
	"fmt"
	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	routeV1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strconv"
)

var ingressAnnotations = map[string]string{
	"kubernetes.io/ingress.class":                "nginx",
	"nginx.ingress.kubernetes.io/rewrite-target": "/",
	"nginx.ingress.kubernetes.io/ssl-redirect":   "false",
}

func GetSpecObjects(spec v1alpha1.WorkspaceRoutingSpec, namespace string) ([]corev1.Service, []v1beta1.Ingress, []routeV1.Route) {
	services := getServicesForSpec(spec, namespace)
	ingresses := getIngressesForSpec(spec, namespace)

	return services, ingresses, nil
}

func getServicesForSpec(spec v1alpha1.WorkspaceRoutingSpec, namespace string) []corev1.Service {
	var servicePorts []corev1.ServicePort
	for _, endpoint := range spec.Endpoints {
		if endpoint.Attributes[v1alpha1.DISCOVERABLE_ATTRIBUTE] != "true" {
			//continue
		}
		servicePorts = append(servicePorts, corev1.ServicePort{
			Name:     endpoint.Name,
			Protocol: corev1.ProtocolTCP, // TODO: use endpoints protocol somehow, but supported set is different?
			Port:     int32(endpoint.Port),
		})
	}
	// TODO: Decide if we _need_ more than one service here?
	return []corev1.Service{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:      "service-" + spec.WorkspaceId, // TODO?
				Namespace: namespace,
				Labels: map[string]string{
					"app": spec.WorkspaceId,
				},
			},
			Spec: corev1.ServiceSpec{
				Ports:    servicePorts,
				Selector: spec.PodSelector,
				Type:     corev1.ServiceTypeClusterIP,
			},
		},
	}
}

func getIngressesForSpec(spec v1alpha1.WorkspaceRoutingSpec, namespace string) []v1beta1.Ingress {
	var ingresses []v1beta1.Ingress
	for _, endpoint := range spec.Endpoints {
		if endpoint.Attributes[v1alpha1.PUBLIC_ENDPOINT_ATTRIBUTE] != "true" {
			continue
		}
		var targetEndpoint intstr.IntOrString
		if endpoint.Name != "" {
			targetEndpoint = intstr.FromString(endpoint.Name)
		} else {
			targetEndpoint = intstr.FromInt(int(endpoint.Port))
		}

		ingresses = append(ingresses, v1beta1.Ingress{
			ObjectMeta: v1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", spec.WorkspaceId, endpoint.Name),
				Namespace: namespace,
				Labels: map[string]string{
					"app": spec.WorkspaceId,
				},
				Annotations: ingressAnnotations,
			},
			Spec: v1beta1.IngressSpec{
				Rules: []v1beta1.IngressRule{
					{
						Host: fmt.Sprintf("%s-%s-%s.%s",
							spec.WorkspaceId, endpoint.Name, strconv.FormatInt(endpoint.Port, 10), spec.IngressGlobalDomain),
						IngressRuleValue: v1beta1.IngressRuleValue{
							HTTP: &v1beta1.HTTPIngressRuleValue{
								Paths: []v1beta1.HTTPIngressPath{
									{
										Backend: v1beta1.IngressBackend{
											ServiceName: "service-" + spec.WorkspaceId, // TODO: Copied from service func above
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
	}
	return ingresses
}

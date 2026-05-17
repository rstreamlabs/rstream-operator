// See LICENSE file in the project root for license information.

package controller

import (
	"fmt"
	"strconv"

	tunnelsv1alpha1 "github.com/rstreamlabs/rstream-operator/api/v1alpha1"
	"github.com/rstreamlabs/rstream-operator/internal/resources"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func serviceNamespace(tunnel *tunnelsv1alpha1.RstreamTunnel) string {
	if tunnel.Spec.Target.Service.Namespace != "" {
		return tunnel.Spec.Target.Service.Namespace
	}
	return tunnel.Namespace
}

func serviceHost(namespace, name string) string {
	return name + "." + namespace + ".svc"
}

func resolveServiceTarget(tunnel *tunnelsv1alpha1.RstreamTunnel, svc *corev1.Service) (resources.ResolvedTarget, error) {
	port, err := resolveServicePort(svc, tunnel.Spec.Target.Service.Port)
	if err != nil {
		return resources.ResolvedTarget{}, err
	}
	expected := expectedServiceProtocol(tunnel)
	actual := port.Protocol
	if actual == "" {
		actual = corev1.ProtocolTCP
	}
	if actual != expected {
		return resources.ResolvedTarget{}, fmt.Errorf("service port %q uses %s but tunnel requires %s", port.Name, actual, expected)
	}
	return resources.ResolvedTarget{
		Host:     serviceHost(svc.Namespace, svc.Name),
		Port:     port.Port,
		Protocol: actual,
	}, nil
}

func resolveServicePort(svc *corev1.Service, desired intstr.IntOrString) (corev1.ServicePort, error) {
	if svc == nil {
		return corev1.ServicePort{}, fmt.Errorf("service is nil")
	}
	for _, port := range svc.Spec.Ports {
		switch desired.Type {
		case intstr.String:
			if port.Name == desired.StrVal {
				return port, nil
			}
		case intstr.Int:
			if port.Port == int32(desired.IntVal) {
				return port, nil
			}
		}
	}
	if desired.Type == intstr.String {
		return corev1.ServicePort{}, fmt.Errorf("service port %q not found", desired.StrVal)
	}
	return corev1.ServicePort{}, fmt.Errorf("service port %s not found", strconv.Itoa(int(desired.IntVal)))
}

func expectedServiceProtocol(tunnel *tunnelsv1alpha1.RstreamTunnel) corev1.Protocol {
	if tunnel.Spec.Type == tunnelsv1alpha1.TunnelTypeDatagram {
		return corev1.ProtocolUDP
	}
	if tunnel.Spec.Type == "" {
		switch tunnel.Spec.Protocol {
		case tunnelsv1alpha1.ProtocolDTLS, tunnelsv1alpha1.ProtocolQUIC:
			return corev1.ProtocolUDP
		case tunnelsv1alpha1.ProtocolHTTP:
			if tunnel.Spec.HTTP != nil && tunnel.Spec.HTTP.Version == tunnelsv1alpha1.HTTPVersion3 {
				return corev1.ProtocolUDP
			}
		}
	}
	return corev1.ProtocolTCP
}

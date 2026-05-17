// See LICENSE file in the project root for license information.

package resources

import (
	tunnelsv1alpha1 "github.com/rstreamlabs/rstream-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type BuildOptions struct {
	AgentImage             string
	ConfigYAML             string
	SecretResourceVersions []string
	Scheme                 *runtime.Scheme
}

func ApplyConfigMap(cm *corev1.ConfigMap, tunnel *tunnelsv1alpha1.RstreamTunnel, opts BuildOptions) error {
	names := NamesFor(tunnel)
	cm.Name = names.ConfigMap
	cm.Namespace = tunnel.Namespace
	cm.Labels = AgentLabels(tunnel, "config")
	cm.Data = map[string]string{ConfigFileName: opts.ConfigYAML}
	if opts.Scheme != nil {
		if err := controllerutil.SetControllerReference(tunnel, cm, opts.Scheme); err != nil {
			return err
		}
	}
	return nil
}

func ApplyDeployment(dep *appsv1.Deployment, tunnel *tunnelsv1alpha1.RstreamTunnel, connection *tunnelsv1alpha1.RstreamConnection, opts BuildOptions) error {
	names := NamesFor(tunnel)
	replicas := int32(1)
	if tunnel.Spec.Agent != nil && tunnel.Spec.Agent.Replicas != nil {
		replicas = *tunnel.Spec.Agent.Replicas
	}
	image := opts.AgentImage
	if tunnel.Spec.Agent != nil && tunnel.Spec.Agent.Image != "" {
		image = tunnel.Spec.Agent.Image
	}
	labels := AgentLabels(tunnel, "agent")
	podLabels := copyMap(labels)
	if tunnel.Spec.Agent != nil {
		for k, v := range tunnel.Spec.Agent.PodLabels {
			podLabels[k] = v
		}
	}
	annotations := map[string]string{
		AnnotationConfigHash: ConfigHash(append([]string{opts.ConfigYAML}, opts.SecretResourceVersions...)...),
	}
	if tunnel.Spec.Agent != nil {
		for k, v := range tunnel.Spec.Agent.PodAnnotations {
			annotations[k] = v
		}
	}
	dep.Name = names.Deployment
	dep.Namespace = tunnel.Namespace
	dep.Labels = labels
	dep.Spec.Replicas = &replicas
	dep.Spec.Strategy = appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType}
	dep.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
	dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
		Labels:      podLabels,
		Annotations: annotations,
	}
	dep.Spec.Template.Spec = corev1.PodSpec{
		ServiceAccountName:           names.ServiceAccount,
		AutomountServiceAccountToken: ptr.To(true),
		SecurityContext: &corev1.PodSecurityContext{
			RunAsNonRoot: ptr.To(true),
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
		},
		Containers: []corev1.Container{{
			Name:            "agent",
			Image:           image,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"/rstream-agent"},
			Args: []string{
				"--config=/etc/rstream-agent/" + ConfigFileName,
				"--health-bind-address=:8081",
			},
			Env:             agentEnv(connection),
			Ports:           []corev1.ContainerPort{{Name: "health", ContainerPort: 8081, Protocol: corev1.ProtocolTCP}},
			VolumeMounts:    agentVolumeMounts(connection),
			Resources:       agentResources(tunnel),
			SecurityContext: agentContainerSecurityContext(),
			LivenessProbe: &corev1.Probe{
				ProbeHandler:        corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: "/healthz", Port: intstrFromInt(8081)}},
				InitialDelaySeconds: 5,
				PeriodSeconds:       10,
				TimeoutSeconds:      2,
				FailureThreshold:    6,
			},
			ReadinessProbe: &corev1.Probe{
				ProbeHandler:     corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: "/readyz", Port: intstrFromInt(8081)}},
				PeriodSeconds:    5,
				TimeoutSeconds:   2,
				FailureThreshold: 3,
			},
		}},
		Volumes: agentVolumes(names, connection),
	}
	if tunnel.Spec.Agent != nil {
		dep.Spec.Template.Spec.NodeSelector = tunnel.Spec.Agent.NodeSelector
		dep.Spec.Template.Spec.Tolerations = tunnel.Spec.Agent.Tolerations
		dep.Spec.Template.Spec.Affinity = tunnel.Spec.Agent.Affinity
		dep.Spec.Template.Spec.PriorityClassName = tunnel.Spec.Agent.PriorityClassName
	}
	if opts.Scheme != nil {
		if err := controllerutil.SetControllerReference(tunnel, dep, opts.Scheme); err != nil {
			return err
		}
	}
	return nil
}

func ApplyServiceAccount(sa *corev1.ServiceAccount, tunnel *tunnelsv1alpha1.RstreamTunnel, opts BuildOptions) error {
	names := NamesFor(tunnel)
	sa.Name = names.ServiceAccount
	sa.Namespace = tunnel.Namespace
	sa.Labels = AgentLabels(tunnel, "agent-rbac")
	if opts.Scheme != nil {
		if err := controllerutil.SetControllerReference(tunnel, sa, opts.Scheme); err != nil {
			return err
		}
	}
	return nil
}

func ApplyRole(role *rbacv1.Role, tunnel *tunnelsv1alpha1.RstreamTunnel, opts BuildOptions) error {
	names := NamesFor(tunnel)
	role.Name = names.Role
	role.Namespace = tunnel.Namespace
	role.Labels = AgentLabels(tunnel, "agent-rbac")
	role.Rules = []rbacv1.PolicyRule{
		{
			APIGroups:     []string{tunnelsv1alpha1.GroupVersion.Group},
			Resources:     []string{"rstreamtunnels"},
			ResourceNames: []string{tunnel.Name},
			Verbs:         []string{"get"},
		},
		{
			APIGroups:     []string{tunnelsv1alpha1.GroupVersion.Group},
			Resources:     []string{"rstreamtunnels/status"},
			ResourceNames: []string{tunnel.Name},
			Verbs:         []string{"get", "patch", "update"},
		},
	}
	if opts.Scheme != nil {
		if err := controllerutil.SetControllerReference(tunnel, role, opts.Scheme); err != nil {
			return err
		}
	}
	return nil
}

func ApplyRoleBinding(rb *rbacv1.RoleBinding, tunnel *tunnelsv1alpha1.RstreamTunnel, opts BuildOptions) error {
	names := NamesFor(tunnel)
	rb.Name = names.RoleBinding
	rb.Namespace = tunnel.Namespace
	rb.Labels = AgentLabels(tunnel, "agent-rbac")
	rb.RoleRef = rbacv1.RoleRef{
		APIGroup: rbacv1.GroupName,
		Kind:     "Role",
		Name:     names.Role,
	}
	rb.Subjects = []rbacv1.Subject{{
		Kind:      rbacv1.ServiceAccountKind,
		Name:      names.ServiceAccount,
		Namespace: tunnel.Namespace,
	}}
	if opts.Scheme != nil {
		if err := controllerutil.SetControllerReference(tunnel, rb, opts.Scheme); err != nil {
			return err
		}
	}
	return nil
}

func agentEnv(connection *tunnelsv1alpha1.RstreamConnection) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name: AgentPodNameEnv,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"},
			},
		},
		{
			Name: AgentPodNamespaceEnv,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"},
			},
		},
	}
	if connection.Spec.TokenSecretRef != nil {
		env = append(env, corev1.EnvVar{
			Name: "RSTREAM_AUTHENTICATION_TOKEN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: connection.Spec.TokenSecretRef.Name},
					Key:                  connection.Spec.TokenSecretRef.Key,
				},
			},
		})
	}
	if connection.Spec.Transport != nil && connection.Spec.Transport.Proxy != nil {
		if connection.Spec.Transport.Proxy.UsernameSecretRef != nil {
			env = append(env, secretEnv(ProxyUsernameEnv, connection.Spec.Transport.Proxy.UsernameSecretRef))
		}
		if connection.Spec.Transport.Proxy.PasswordSecretRef != nil {
			env = append(env, secretEnv(ProxyPasswordEnv, connection.Spec.Transport.Proxy.PasswordSecretRef))
		}
	}
	return env
}

func secretEnv(name string, ref *tunnelsv1alpha1.SecretKeyRef) corev1.EnvVar {
	return corev1.EnvVar{
		Name: name,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: ref.Name},
				Key:                  ref.Key,
			},
		},
	}
}

func agentVolumes(names Names, connection *tunnelsv1alpha1.RstreamConnection) []corev1.Volume {
	volumes := []corev1.Volume{{
		Name: "config",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: names.ConfigMap},
				Items: []corev1.KeyToPath{{
					Key:  ConfigFileName,
					Path: ConfigFileName,
				}},
			},
		},
	}}
	if connection.Spec.MTLS != nil {
		sources := []corev1.VolumeProjection{
			secretProjection(connection.Spec.MTLS.CertSecretRef, "tls.crt"),
			secretProjection(connection.Spec.MTLS.KeySecretRef, "tls.key"),
		}
		if connection.Spec.MTLS.CASecretRef != nil {
			sources = append(sources, secretProjection(*connection.Spec.MTLS.CASecretRef, "ca.crt"))
		}
		volumes = append(volumes, corev1.Volume{
			Name: "mtls",
			VolumeSource: corev1.VolumeSource{
				Projected: &corev1.ProjectedVolumeSource{Sources: sources},
			},
		})
	}
	return volumes
}

func secretProjection(ref tunnelsv1alpha1.SecretKeyRef, path string) corev1.VolumeProjection {
	return corev1.VolumeProjection{
		Secret: &corev1.SecretProjection{
			LocalObjectReference: corev1.LocalObjectReference{Name: ref.Name},
			Items: []corev1.KeyToPath{{
				Key:  ref.Key,
				Path: path,
			}},
		},
	}
}

func agentVolumeMounts(connection *tunnelsv1alpha1.RstreamConnection) []corev1.VolumeMount {
	mounts := []corev1.VolumeMount{{
		Name:      "config",
		MountPath: "/etc/rstream-agent",
		ReadOnly:  true,
	}}
	if connection.Spec.MTLS != nil {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      "mtls",
			MountPath: "/var/run/rstream/mtls",
			ReadOnly:  true,
		})
	}
	return mounts
}

func agentResources(tunnel *tunnelsv1alpha1.RstreamTunnel) corev1.ResourceRequirements {
	if tunnel.Spec.Agent != nil && !isZeroResources(tunnel.Spec.Agent.Resources) {
		return tunnel.Spec.Agent.Resources
	}
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("25m"),
			corev1.ResourceMemory: resource.MustParse("64Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("256Mi"),
		},
	}
}

func isZeroResources(resources corev1.ResourceRequirements) bool {
	return len(resources.Claims) == 0 && len(resources.Limits) == 0 && len(resources.Requests) == 0
}

func agentContainerSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		AllowPrivilegeEscalation: ptr.To(false),
		ReadOnlyRootFilesystem:   ptr.To(true),
		RunAsNonRoot:             ptr.To(true),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
	}
}

func intstrFromInt(value int) intstr.IntOrString {
	return intstr.FromInt(value)
}

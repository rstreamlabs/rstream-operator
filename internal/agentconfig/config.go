// See LICENSE file in the project root for license information.

package agentconfig

// Config is the non-secret configuration consumed by rstream-agent.
type Config struct {
	Connection ConnectionConfig `yaml:"connection"`
	Tunnel     TunnelConfig     `yaml:"tunnel"`
	Target     TargetConfig     `yaml:"target"`
	Kubernetes KubernetesConfig `yaml:"kubernetes"`
	Retry      RetryConfig      `yaml:"retry,omitempty"`
}

type ConnectionConfig struct {
	Engine    string           `yaml:"engine"`
	MTLS      *MTLSConfig      `yaml:"mtls,omitempty"`
	Transport *TransportConfig `yaml:"transport,omitempty"`
}

type MTLSConfig struct {
	CertFile string `yaml:"certFile"`
	KeyFile  string `yaml:"keyFile"`
	CAFile   string `yaml:"caFile,omitempty"`
}

type TransportConfig struct {
	Bind     *BindConfig  `yaml:"bind,omitempty"`
	IPFamily string       `yaml:"ipFamily,omitempty"`
	DNS      *DNSConfig   `yaml:"dns,omitempty"`
	MPTCP    *bool        `yaml:"mptcp,omitempty"`
	Proxy    *ProxyConfig `yaml:"proxy,omitempty"`
	UseQUIC  *bool        `yaml:"useQuic,omitempty"`
}

type BindConfig struct {
	Mode      string `yaml:"mode,omitempty"`
	Interface string `yaml:"interface,omitempty"`
	Address   string `yaml:"address,omitempty"`
}

type DNSConfig struct {
	Override   string `yaml:"override,omitempty"`
	TLS        *bool  `yaml:"tls,omitempty"`
	ServerName string `yaml:"serverName,omitempty"`
	DNSSEC     *bool  `yaml:"dnssec,omitempty"`
}

type ProxyConfig struct {
	HTTP        string            `yaml:"http,omitempty"`
	UsernameEnv string            `yaml:"usernameEnv,omitempty"`
	PasswordEnv string            `yaml:"passwordEnv,omitempty"`
	Headers     map[string]string `yaml:"headers,omitempty"`
}

type TunnelConfig struct {
	Name        string            `yaml:"name"`
	Publish     *bool             `yaml:"publish,omitempty"`
	Protocol    string            `yaml:"protocol,omitempty"`
	Type        string            `yaml:"type,omitempty"`
	Hostname    string            `yaml:"hostname,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	UpstreamTLS *bool             `yaml:"upstreamTLS,omitempty"`
	TrustedIPs  []string          `yaml:"trustedIPs,omitempty"`
	GeoIP       []string          `yaml:"geoip,omitempty"`
	HTTP        *HTTPConfig       `yaml:"http,omitempty"`
	TLS         *TLSConfig        `yaml:"tls,omitempty"`
}

type HTTPConfig struct {
	Version string          `yaml:"version,omitempty"`
	Auth    *HTTPAuthConfig `yaml:"auth,omitempty"`
	Gate    *HTTPGateConfig `yaml:"gate,omitempty"`
}

type HTTPAuthConfig struct {
	Token   *bool `yaml:"token,omitempty"`
	Rstream *bool `yaml:"rstream,omitempty"`
}

type HTTPGateConfig struct {
	Challenge *bool `yaml:"challenge,omitempty"`
}

type TLSConfig struct {
	Mode       string   `yaml:"mode,omitempty"`
	MinVersion string   `yaml:"minVersion,omitempty"`
	ALPNs      []string `yaml:"alpns,omitempty"`
	MTLS       *bool    `yaml:"mtls,omitempty"`
}

type TargetConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Protocol string `yaml:"protocol"`
}

type KubernetesConfig struct {
	Enabled         bool   `yaml:"enabled"`
	Namespace       string `yaml:"namespace,omitempty"`
	TunnelName      string `yaml:"tunnelName,omitempty"`
	PodNameEnv      string `yaml:"podNameEnv,omitempty"`
	PodNamespaceEnv string `yaml:"podNamespaceEnv,omitempty"`
}

type RetryConfig struct {
	Initial string `yaml:"initial,omitempty"`
	Max     string `yaml:"max,omitempty"`
}

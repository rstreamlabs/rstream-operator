// See LICENSE file in the project root for license information.

package agent

import (
	"fmt"
	"strings"

	"github.com/rstreamlabs/rstream-go"
	"github.com/rstreamlabs/rstream-operator/internal/agentconfig"
)

func TunnelProperties(cfg agentconfig.TunnelConfig) (rstream.TunnelProperties, error) {
	props := rstream.TunnelProperties{}
	name := strings.TrimSpace(cfg.Name)
	if name == "" {
		return props, fmt.Errorf("tunnel name is required")
	}
	props.Name = &name
	if cfg.Publish != nil {
		props.Publish = cfg.Publish
	}
	if strings.TrimSpace(cfg.Type) != "" {
		val, err := parseTunnelType(cfg.Type)
		if err != nil {
			return props, err
		}
		props.Type = &val
	}
	if strings.TrimSpace(cfg.Protocol) != "" {
		val, err := parseProtocol(cfg.Protocol)
		if err != nil {
			return props, err
		}
		props.Protocol = &val
	}
	if strings.TrimSpace(cfg.Hostname) != "" {
		hostname := strings.TrimSpace(cfg.Hostname)
		props.Hostname = &hostname
	}
	if len(cfg.Labels) > 0 {
		props.Labels = copyMap(cfg.Labels)
	}
	if len(cfg.GeoIP) > 0 {
		props.GeoIP = append([]string(nil), cfg.GeoIP...)
	}
	if len(cfg.TrustedIPs) > 0 {
		props.TrustedIPs = append([]string(nil), cfg.TrustedIPs...)
	}
	if cfg.UpstreamTLS != nil {
		props.UpstreamTLS = cfg.UpstreamTLS
		if props.Protocol == nil || *props.Protocol == rstream.ProtocolHTTP {
			props.HTTPUseTLS = cfg.UpstreamTLS
		}
	}
	if cfg.HTTP != nil {
		if props.Protocol != nil && *props.Protocol != rstream.ProtocolHTTP {
			return props, fmt.Errorf("http settings require protocol %q", rstream.ProtocolHTTP)
		}
		if strings.TrimSpace(cfg.HTTP.Version) != "" {
			val, err := parseHTTPVersion(cfg.HTTP.Version)
			if err != nil {
				return props, err
			}
			props.HTTPVersion = &val
		}
		if cfg.HTTP.Auth != nil {
			if cfg.HTTP.Auth.Token != nil {
				props.TokenAuth = cfg.HTTP.Auth.Token
			}
			if cfg.HTTP.Auth.Rstream != nil {
				props.RstreamAuth = cfg.HTTP.Auth.Rstream
			}
		}
		if cfg.HTTP.Gate != nil && cfg.HTTP.Gate.Challenge != nil {
			props.ChallengeMode = cfg.HTTP.Gate.Challenge
		}
	}
	if cfg.TLS != nil {
		if strings.TrimSpace(cfg.TLS.Mode) != "" {
			val, err := parseTLSMode(cfg.TLS.Mode)
			if err != nil {
				return props, err
			}
			props.TLSMode = &val
		}
		if strings.TrimSpace(cfg.TLS.MinVersion) != "" {
			val, err := parseTLSMinVersion(cfg.TLS.MinVersion)
			if err != nil {
				return props, err
			}
			props.TLSMinVersion = &val
		}
		if len(cfg.TLS.ALPNs) > 0 {
			props.TLSALPNs = append([]string(nil), cfg.TLS.ALPNs...)
		}
		if cfg.TLS.MTLS != nil {
			props.MTLSAuth = cfg.TLS.MTLS
		}
	}
	return props, nil
}

func parseTunnelType(val string) (rstream.TunnelType, error) {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case string(rstream.TunnelTypeBytestream):
		return rstream.TunnelTypeBytestream, nil
	case string(rstream.TunnelTypeDatagram):
		return rstream.TunnelTypeDatagram, nil
	default:
		return "", fmt.Errorf("invalid tunnel type %q", val)
	}
}

func parseProtocol(val string) (rstream.Protocol, error) {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case string(rstream.ProtocolHTTP):
		return rstream.ProtocolHTTP, nil
	case string(rstream.ProtocolTLS):
		return rstream.ProtocolTLS, nil
	case string(rstream.ProtocolDTLS):
		return rstream.ProtocolDTLS, nil
	case string(rstream.ProtocolQUIC):
		return rstream.ProtocolQUIC, nil
	default:
		return "", fmt.Errorf("invalid protocol %q", val)
	}
}

func parseHTTPVersion(val string) (rstream.HTTPVersion, error) {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case string(rstream.HTTP1_1):
		return rstream.HTTP1_1, nil
	case string(rstream.HTTP2):
		return rstream.HTTP2, nil
	case string(rstream.HTTP3):
		return rstream.HTTP3, nil
	default:
		return "", fmt.Errorf("invalid http version %q", val)
	}
}

func parseTLSMode(val string) (rstream.TLSMode, error) {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case string(rstream.TLSModeTerminated):
		return rstream.TLSModeTerminated, nil
	case string(rstream.TLSModePassthrough):
		return rstream.TLSModePassthrough, nil
	default:
		return "", fmt.Errorf("invalid tls mode %q", val)
	}
}

func parseTLSMinVersion(val string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "tls1.2", "tls1.3":
		return strings.ToLower(strings.TrimSpace(val)), nil
	default:
		return "", fmt.Errorf("invalid tls min version %q", val)
	}
}

func copyMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

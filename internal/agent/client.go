// See LICENSE file in the project root for license information.

package agent

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	"github.com/rstreamlabs/rstream-go"
	rstreamconfig "github.com/rstreamlabs/rstream-go/config"
	"github.com/rstreamlabs/rstream-operator/internal/agentconfig"
)

const tokenEnv = "RSTREAM_AUTHENTICATION_TOKEN"

func NewRstreamClient(cfg agentconfig.Config) (*rstream.Client, error) {
	tlsConfig, err := tlsConfig(cfg.Connection.MTLS)
	if err != nil {
		return nil, err
	}
	token := strings.TrimSpace(os.Getenv(tokenEnv))
	if token == "" && tlsConfig == nil {
		return nil, fmt.Errorf("%s is required when mTLS is not configured", tokenEnv)
	}
	if token != "" && tlsConfig != nil {
		return nil, fmt.Errorf("%s and mTLS authentication cannot be used together", tokenEnv)
	}
	return rstream.NewClient(rstream.ClientOptions{
		Engine:          cfg.Connection.Engine,
		Token:           token,
		TLSClientConfig: tlsConfig,
		Transport:       transport(cfg.Connection.Transport),
	})
}

func tlsConfig(cfg *agentconfig.MTLSConfig) (*tls.Config, error) {
	if cfg == nil {
		return nil, nil
	}
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load mTLS key pair: %w", err)
	}
	out := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}
	if strings.TrimSpace(cfg.CAFile) != "" {
		pemBytes, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read mTLS CA bundle: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pemBytes) {
			return nil, fmt.Errorf("mTLS CA bundle does not contain any PEM certificates")
		}
		out.RootCAs = pool
	}
	return out, nil
}

func transport(cfg *agentconfig.TransportConfig) rstream.Dialer {
	if cfg == nil {
		return nil
	}
	converted := &rstreamconfig.TransportConfig{
		IPFamily: strings.TrimSpace(cfg.IPFamily),
		MPTCP:    cfg.MPTCP,
		UseQUIC:  cfg.UseQUIC,
	}
	if converted.IPFamily == "auto" {
		converted.IPFamily = ""
	}
	if cfg.Bind != nil {
		converted.Bind = &rstreamconfig.BindConfig{
			Mode:      cfg.Bind.Mode,
			Interface: cfg.Bind.Interface,
			Address:   cfg.Bind.Address,
		}
	}
	if cfg.DNS != nil {
		converted.DNS = &rstreamconfig.DNSConfig{
			Override:   cfg.DNS.Override,
			TLS:        cfg.DNS.TLS,
			ServerName: cfg.DNS.ServerName,
			DNSSEC:     cfg.DNS.DNSSEC,
		}
	}
	if cfg.Proxy != nil {
		converted.Proxy = &rstreamconfig.ProxyConfig{
			HTTP:    cfg.Proxy.HTTP,
			Headers: copyMap(cfg.Proxy.Headers),
		}
		if cfg.Proxy.UsernameEnv != "" {
			converted.Proxy.Username = os.Getenv(cfg.Proxy.UsernameEnv)
		}
		if cfg.Proxy.PasswordEnv != "" {
			converted.Proxy.Password = os.Getenv(cfg.Proxy.PasswordEnv)
		}
	}
	return rstreamconfig.FlattenTransport(converted)
}

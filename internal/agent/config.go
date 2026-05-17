// See LICENSE file in the project root for license information.

package agent

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rstreamlabs/rstream-operator/internal/agentconfig"
	"gopkg.in/yaml.v3"
)

const (
	DefaultRetryInitial = time.Second
	DefaultRetryMax     = 30 * time.Second
)

func LoadConfig(path string) (agentconfig.Config, error) {
	var cfg agentconfig.Config
	if strings.TrimSpace(path) == "" {
		return cfg, errors.New("config path is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return cfg, fmt.Errorf("decode config: %w", err)
	}
	if err := ValidateConfig(cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func ValidateConfig(cfg agentconfig.Config) error {
	if strings.TrimSpace(cfg.Connection.Engine) == "" {
		return errors.New("connection.engine is required")
	}
	if strings.TrimSpace(cfg.Tunnel.Name) == "" {
		return errors.New("tunnel.name is required")
	}
	if strings.TrimSpace(cfg.Target.Host) == "" {
		return errors.New("target.host is required")
	}
	if strings.TrimSpace(cfg.Target.Port) == "" {
		return errors.New("target.port is required")
	}
	if cfg.Connection.MTLS != nil {
		if strings.TrimSpace(cfg.Connection.MTLS.CertFile) == "" {
			return errors.New("connection.mtls.certFile is required")
		}
		if strings.TrimSpace(cfg.Connection.MTLS.KeyFile) == "" {
			return errors.New("connection.mtls.keyFile is required")
		}
	}
	if cfg.Kubernetes.Enabled {
		if strings.TrimSpace(cfg.Kubernetes.Namespace) == "" {
			return errors.New("kubernetes.namespace is required when kubernetes.enabled is true")
		}
		if strings.TrimSpace(cfg.Kubernetes.TunnelName) == "" {
			return errors.New("kubernetes.tunnelName is required when kubernetes.enabled is true")
		}
	}
	return nil
}

func RetryDurations(cfg agentconfig.Config) (time.Duration, time.Duration, error) {
	initial := DefaultRetryInitial
	maximum := DefaultRetryMax
	var err error
	if strings.TrimSpace(cfg.Retry.Initial) != "" {
		initial, err = time.ParseDuration(cfg.Retry.Initial)
		if err != nil {
			return 0, 0, fmt.Errorf("retry.initial: %w", err)
		}
	}
	if strings.TrimSpace(cfg.Retry.Max) != "" {
		maximum, err = time.ParseDuration(cfg.Retry.Max)
		if err != nil {
			return 0, 0, fmt.Errorf("retry.max: %w", err)
		}
	}
	if initial <= 0 {
		return 0, 0, errors.New("retry.initial must be greater than zero")
	}
	if maximum < initial {
		return 0, 0, errors.New("retry.max must be greater than or equal to retry.initial")
	}
	return initial, maximum, nil
}

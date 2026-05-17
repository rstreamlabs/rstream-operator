// See LICENSE file in the project root for license information.

package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	tunnelsv1alpha1 "github.com/rstreamlabs/rstream-operator/api/v1alpha1"
	"github.com/rstreamlabs/rstream-operator/internal/agent"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(tunnelsv1alpha1.AddToScheme(scheme))
}

func main() {
	var configPath string
	var healthBindAddress string
	flag.StringVar(&configPath, "config", "/etc/rstream-agent/config.yaml", "Path to the agent configuration file")
	flag.StringVar(&healthBindAddress, "health-bind-address", ":8081", "The address the health endpoint binds to")
	flag.Parse()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	cfg, err := agent.LoadConfig(configPath)
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	health := &agent.HealthState{}
	go func() {
		if err := agent.RunHealthServer(ctx, healthBindAddress, health, logger.With("component", "health")); err != nil {
			logger.Error("Health server failed", "error", err)
			stop()
		}
	}()
	reporter := agent.Reporter(agent.NoopReporter{})
	if cfg.Kubernetes.Enabled {
		k8sClient, err := client.New(ctrl.GetConfigOrDie(), client.Options{Scheme: scheme})
		if err != nil {
			logger.Error("Failed to create Kubernetes client", "error", err)
			os.Exit(1)
		}
		reporter = agent.NewKubernetesReporter(k8sClient, cfg.Kubernetes)
	}
	runner := agent.Runner{
		Config:   cfg,
		Reporter: reporter,
		Health:   health,
		Logger:   logger.With("component", "runner"),
	}
	if err := runner.Run(ctx); err != nil {
		logger.Error("Agent failed", "error", err)
		os.Exit(1)
	}
}

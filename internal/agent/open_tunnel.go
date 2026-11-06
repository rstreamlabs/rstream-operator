// See LICENSE file in the project root for license information.

package agent

import (
	"context"
	"errors"
	"strings"

	"github.com/rstreamlabs/rstream-go"
	"github.com/rstreamlabs/rstream-operator/internal/agentconfig"
)

func openTunnel(ctx context.Context, ctrl rstream.ControlChannel, props rstream.TunnelProperties, connection agentconfig.ConnectionConfig) (rstream.Tunnel, error) {
	if connection.StableDomainEngine != "" {
		// A project can publish on its global domain while the agent dials a region.
		if err := rstream.MaybeSetGeneratedStableDomain(&props, connection.StableDomainEngine); err != nil {
			return nil, err
		}
		return ctrl.CreateTunnel(ctx, props)
	}

	// A direct Engine connection has no Control plane metadata. Preserve explicit
	// names and use the SDK's project-scoped slug when the address supports one.
	candidate := props
	if err := rstream.MaybeSetGeneratedStableDomain(&candidate, connection.Engine); err != nil {
		return nil, err
	}
	tunnel, err := ctrl.CreateTunnel(ctx, props)
	if err != nil || props.Hostname != nil || candidate.Hostname == nil {
		return tunnel, err
	}

	// An Engine-allocated hostname is temporary in EE. Read its serving domain,
	// then release the tunnel before requesting the stable name (also at quota 1).
	assigned, readErr := tunnel.Properties()
	closeErr := tunnel.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	_, domain, ok := strings.Cut(stringPtrValue(assigned.Hostname), ".t.")
	if !ok || domain == "" {
		return nil, errors.New("engine returned an allocated tunnel without a serving domain")
	}
	label, _, _ := strings.Cut(*candidate.Hostname, ".")
	hostname := label + ".t." + domain
	props.Hostname = &hostname
	return ctrl.CreateTunnel(ctx, props)
}

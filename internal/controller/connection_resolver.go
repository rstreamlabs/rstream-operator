// See LICENSE file in the project root for license information.

package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/rstreamlabs/rstream-go/controlplane"
	tunnelsv1alpha1 "github.com/rstreamlabs/rstream-operator/api/v1alpha1"
)

const defaultAPIURL = "https://rstream.io"

type connectionResolution struct {
	Engine          string
	APIURL          string
	ProjectID       string
	ProjectEndpoint string
	Region          string
}

type connectionResolver interface {
	Resolve(ctx context.Context, connection *tunnelsv1alpha1.RstreamConnection, token string, headers map[string]string) (connectionResolution, error)
}

type defaultConnectionResolver struct{}

func (defaultConnectionResolver) Resolve(ctx context.Context, connection *tunnelsv1alpha1.RstreamConnection, token string, headers map[string]string) (connectionResolution, error) {
	if connection == nil {
		return connectionResolution{}, errors.New("RstreamConnection is nil")
	}
	if engine := strings.TrimSpace(connection.Spec.Engine); engine != "" {
		if strings.TrimSpace(connection.Spec.Region) != "" {
			return connectionResolution{}, errors.New("region cannot be used with an explicit engine")
		}
		if len(headers) > 0 {
			return connectionResolution{}, errors.New("controlPlaneHeaders cannot be used with an explicit engine")
		}
		return connectionResolution{Engine: engine}, nil
	}
	apiURL := connectionAPIURL(connection)
	if strings.TrimSpace(token) == "" {
		return connectionResolution{}, errors.New("tokenSecretRef is required when projectEndpoint or projectID is used")
	}
	client := controlplane.NewClient(apiURL, token, controlplane.WithHeaders(headers))
	if endpoint := strings.TrimSpace(connection.Spec.ProjectEndpoint); endpoint != "" {
		project, err := client.ResolveProjectByEndpoint(ctx, endpoint)
		if err != nil {
			return connectionResolution{}, fmt.Errorf("resolve project endpoint %q: %w", endpoint, err)
		}
		return resolutionFromProject(apiURL, connection.Spec.Region, project)
	}
	if projectID := strings.TrimSpace(connection.Spec.ProjectID); projectID != "" {
		project, err := client.ResolveProjectByID(ctx, projectID)
		if err != nil {
			return connectionResolution{}, fmt.Errorf("resolve project ID %q: %w", projectID, err)
		}
		return resolutionFromProject(apiURL, connection.Spec.Region, project)
	}
	return connectionResolution{}, errors.New("one of projectEndpoint, projectID, or engine is required")
}

func connectionAPIURL(connection *tunnelsv1alpha1.RstreamConnection) string {
	if connection == nil {
		return defaultAPIURL
	}
	if apiURL := strings.TrimRight(strings.TrimSpace(connection.Spec.APIURL), "/"); apiURL != "" {
		return apiURL
	}
	return defaultAPIURL
}

func resolutionFromProject(apiURL, region string, project controlplane.Project) (connectionResolution, error) {
	normalizedRegion, err := controlplane.NormalizeRegion(region)
	if err != nil {
		return connectionResolution{}, err
	}
	engine, err := project.EngineAddressForRegion(normalizedRegion)
	if err != nil {
		return connectionResolution{}, err
	}
	if strings.TrimSpace(engine) == "" {
		return connectionResolution{}, errors.New("resolved project does not include an engine address")
	}
	selectedRegion := normalizedRegion
	if selectedRegion == "" {
		selectedRegion = "auto"
	}
	return connectionResolution{
		Engine:          engine,
		APIURL:          apiURL,
		ProjectID:       project.ID,
		ProjectEndpoint: project.Endpoint,
		Region:          selectedRegion,
	}, nil
}

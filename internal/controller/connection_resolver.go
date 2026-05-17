// See LICENSE file in the project root for license information.

package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rstreamlabs/rstream-go/controlplane"
	tunnelsv1alpha1 "github.com/rstreamlabs/rstream-operator/api/v1alpha1"
)

const defaultAPIURL = "https://rstream.io"

var controlPlaneHTTPClient = &http.Client{Timeout: 10 * time.Second}

type connectionResolution struct {
	Engine          string
	APIURL          string
	ProjectID       string
	ProjectEndpoint string
}

type connectionResolver interface {
	Resolve(ctx context.Context, connection *tunnelsv1alpha1.RstreamConnection, token string) (connectionResolution, error)
}

type defaultConnectionResolver struct{}

func (defaultConnectionResolver) Resolve(ctx context.Context, connection *tunnelsv1alpha1.RstreamConnection, token string) (connectionResolution, error) {
	if connection == nil {
		return connectionResolution{}, errors.New("RstreamConnection is nil")
	}
	if engine := strings.TrimSpace(connection.Spec.Engine); engine != "" {
		return connectionResolution{Engine: engine}, nil
	}
	apiURL := connectionAPIURL(connection)
	if strings.TrimSpace(token) == "" {
		return connectionResolution{}, errors.New("tokenSecretRef is required when projectEndpoint or projectID is used")
	}
	if endpoint := strings.TrimSpace(connection.Spec.ProjectEndpoint); endpoint != "" {
		project, err := controlplane.NewClient(apiURL, token).ResolveProjectByEndpoint(ctx, endpoint)
		if err != nil {
			return connectionResolution{}, fmt.Errorf("resolve project endpoint %q: %w", endpoint, err)
		}
		return resolutionFromProject(apiURL, project)
	}
	if projectID := strings.TrimSpace(connection.Spec.ProjectID); projectID != "" {
		project, err := resolveProjectByID(ctx, apiURL, token, projectID)
		if err != nil {
			return connectionResolution{}, fmt.Errorf("resolve project ID %q: %w", projectID, err)
		}
		return resolutionFromProject(apiURL, project)
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

func resolutionFromProject(apiURL string, project controlplane.Project) (connectionResolution, error) {
	engine := project.EngineAddress()
	if strings.TrimSpace(engine) == "" {
		return connectionResolution{}, errors.New("resolved project does not include an engine address")
	}
	return connectionResolution{
		Engine:          engine,
		APIURL:          apiURL,
		ProjectID:       project.ID,
		ProjectEndpoint: project.Endpoint,
	}, nil
}

func resolveProjectByID(ctx context.Context, apiURL, token, projectID string) (controlplane.Project, error) {
	fullURL := strings.TrimRight(strings.TrimSpace(apiURL), "/") + "/api/projects/tunnels/" + url.PathEscape(projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return controlplane.Project{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := controlPlaneHTTPClient.Do(req)
	if err != nil {
		return controlplane.Project{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = resp.Status
		}
		return controlplane.Project{}, errors.New(message)
	}
	var project controlplane.Project
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return controlplane.Project{}, err
	}
	return project, nil
}

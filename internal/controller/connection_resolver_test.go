// See LICENSE file in the project root for license information.

package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	tunnelsv1alpha1 "github.com/rstreamlabs/rstream-operator/api/v1alpha1"
)

func TestDefaultConnectionResolverUsesManualEngine(t *testing.T) {
	connection := &tunnelsv1alpha1.RstreamConnection{
		Spec: tunnelsv1alpha1.RstreamConnectionSpec{Engine: "engine.example.com:443"},
	}
	resolution, err := (defaultConnectionResolver{}).Resolve(context.Background(), connection, "")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolution.Engine != "engine.example.com:443" {
		t.Fatalf("Engine = %q", resolution.Engine)
	}
}

func TestDefaultConnectionResolverResolvesProjectID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/api/projects/tunnels/project%2Fid" {
			t.Fatalf("path = %q", r.URL.EscapedPath())
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("Authorization = %q", got)
		}
		_, _ = w.Write([]byte(`{"id":"project/id","endpoint":"abc12345","domain":"cluster.example.com","enginePort":443}`))
	}))
	defer server.Close()
	connection := &tunnelsv1alpha1.RstreamConnection{
		Spec: tunnelsv1alpha1.RstreamConnectionSpec{
			APIURL:    server.URL,
			ProjectID: "project/id",
		},
	}
	resolution, err := (defaultConnectionResolver{}).Resolve(context.Background(), connection, "token")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolution.Engine != "abc12345.cluster.example.com:443" {
		t.Fatalf("Engine = %q", resolution.Engine)
	}
	if resolution.ProjectEndpoint != "abc12345" || resolution.ProjectID != "project/id" {
		t.Fatalf("unexpected project fields: %#v", resolution)
	}
}

func TestDefaultConnectionResolverRequiresTokenForProjectLookup(t *testing.T) {
	connection := &tunnelsv1alpha1.RstreamConnection{
		Spec: tunnelsv1alpha1.RstreamConnectionSpec{ProjectEndpoint: "abc12345"},
	}
	if _, err := (defaultConnectionResolver{}).Resolve(context.Background(), connection, ""); err == nil {
		t.Fatalf("Resolve() error = nil, want token error")
	}
}

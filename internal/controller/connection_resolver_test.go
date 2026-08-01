// See LICENSE file in the project root for license information.

package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tunnelsv1alpha1 "github.com/rstreamlabs/rstream-operator/api/v1alpha1"
)

func TestDefaultConnectionResolverUsesManualEngine(t *testing.T) {
	connection := &tunnelsv1alpha1.RstreamConnection{
		Spec: tunnelsv1alpha1.RstreamConnectionSpec{Engine: "engine.example.com:443"},
	}
	resolution, err := (defaultConnectionResolver{}).Resolve(context.Background(), connection, "", nil)
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
		if got := r.Header.Get("x-deployment-bypass"); got != "secret" {
			t.Fatalf("x-deployment-bypass = %q", got)
		}
		_, _ = w.Write([]byte(`{"id":"project/id","endpoint":"abc12345","routing":"global","domain":"global.example.com","enginePort":443,"regionalEndpoints":[{"provider":"aws","region":"us-east-1","domain":"us.example.com","enginePort":8443}]}`))
	}))
	defer server.Close()
	connection := &tunnelsv1alpha1.RstreamConnection{
		Spec: tunnelsv1alpha1.RstreamConnectionSpec{
			APIURL:    server.URL,
			ProjectID: "project/id",
			Region:    "US-EAST-1",
		},
	}
	resolution, err := (defaultConnectionResolver{}).Resolve(context.Background(), connection, "token", map[string]string{"x-deployment-bypass": "secret"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolution.Engine != "abc12345.us.example.com:8443" {
		t.Fatalf("Engine = %q", resolution.Engine)
	}
	if resolution.Region != "us-east-1" {
		t.Fatalf("Region = %q", resolution.Region)
	}
	if resolution.ProjectEndpoint != "abc12345" || resolution.ProjectID != "project/id" {
		t.Fatalf("unexpected project fields: %#v", resolution)
	}
}

func TestDefaultConnectionResolverRequiresTokenForProjectLookup(t *testing.T) {
	connection := &tunnelsv1alpha1.RstreamConnection{
		Spec: tunnelsv1alpha1.RstreamConnectionSpec{ProjectEndpoint: "abc12345"},
	}
	if _, err := (defaultConnectionResolver{}).Resolve(context.Background(), connection, "", nil); err == nil {
		t.Fatalf("Resolve() error = nil, want token error")
	}
}

func TestDefaultConnectionResolverRejectsRegionForManualEngine(t *testing.T) {
	connection := &tunnelsv1alpha1.RstreamConnection{Spec: tunnelsv1alpha1.RstreamConnectionSpec{Engine: "engine.example.com:443", Region: "eu-west-3"}}
	if _, err := (defaultConnectionResolver{}).Resolve(context.Background(), connection, "", nil); err == nil {
		t.Fatal("Resolve() error = nil, want region error")
	}
}

func TestDefaultConnectionResolverRejectsControlPlaneHeadersForManualEngine(t *testing.T) {
	connection := &tunnelsv1alpha1.RstreamConnection{Spec: tunnelsv1alpha1.RstreamConnectionSpec{Engine: "engine.example.com:443"}}
	if _, err := (defaultConnectionResolver{}).Resolve(context.Background(), connection, "", map[string]string{"x-deployment-bypass": "secret"}); err == nil {
		t.Fatal("Resolve() error = nil, want control plane header error")
	}
}

func TestDefaultConnectionResolverRejectsInvalidControlPlaneHeadersBeforeNetworkIO(t *testing.T) {
	connection := &tunnelsv1alpha1.RstreamConnection{
		Spec: tunnelsv1alpha1.RstreamConnectionSpec{
			APIURL:          "http://127.0.0.1:1",
			ProjectEndpoint: "abc12345",
		},
	}
	tests := []struct {
		name    string
		headers map[string]string
		want    string
	}{
		{name: "reserved", headers: map[string]string{"Authorization": "secret"}, want: "reserved control plane header"},
		{name: "duplicate", headers: map[string]string{"X-Access": "one", "x-access": "two"}, want: "duplicate control plane header"},
		{name: "invalid value", headers: map[string]string{"X-Access": "one\ntwo"}, want: "invalid value for control plane header"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := (defaultConnectionResolver{}).Resolve(context.Background(), connection, "token", test.headers)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Resolve() error = %v, want %q", err, test.want)
			}
		})
	}
}

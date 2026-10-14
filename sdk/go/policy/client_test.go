package policy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveConnect(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b := make([]byte, r.ContentLength)
		if r.ContentLength > 0 {
			_, _ = r.Body.Read(b)
		}
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"decision":{"intercept":true,"intercept_reason":"policy_rule"},"revision":"rev-1"}`))
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, srv.Client())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, err := c.ResolveConnect(context.Background(), ConnectRequest{AgentToken: "at", Host: "example.com"})
	if err != nil {
		t.Fatalf("ResolveConnect: %v", err)
	}
	if !resp.Decision.Intercept || resp.Revision != "rev-1" {
		t.Errorf("resp = %+v", resp)
	}
	if gotPath != ConnectPath {
		t.Errorf("path = %q", gotPath)
	}
	var req ConnectRequest
	_ = json.Unmarshal([]byte(gotBody), &req)
	if req.AgentToken != "at" || req.Host != "example.com" {
		t.Errorf("request = %+v", req)
	}
}

func TestRevision(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != RevisionPath || r.URL.Query().Get("agent_token") != "at" {
			t.Errorf("unexpected request %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"revision":"rev-9"}`))
	}))
	defer srv.Close()
	c, _ := NewClient(srv.URL, srv.Client())
	rev, err := c.Revision(context.Background(), "at")
	if err != nil || rev != "rev-9" {
		t.Fatalf("rev=%q err=%v", rev, err)
	}
}

func TestNewClientValidation(t *testing.T) {
	if _, err := NewClient("", nil); err == nil {
		t.Fatal("expected error for empty base URL")
	}
}

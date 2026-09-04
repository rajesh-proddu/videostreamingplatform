package metadata

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientDisabledWithoutBaseURL(t *testing.T) {
	c := NewClient("")
	if c.Enabled() {
		t.Fatal("client with empty baseURL must report disabled")
	}
	if err := c.MarkUploadComplete(context.Background(), "vid-1"); err == nil {
		t.Fatal("expected an error when the client is not configured")
	}

	// dataservice calls this on a possibly-nil client; it must not panic.
	var nilClient *Client
	if nilClient.Enabled() {
		t.Fatal("nil client must report disabled")
	}
}

func TestMarkUploadCompleteSendsStatusUpdate(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if err := NewClient(srv.URL).MarkUploadComplete(context.Background(), "vid-1"); err != nil {
		t.Fatalf("mark complete: %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Errorf("method = %s, want PUT", gotMethod)
	}
	if gotPath != "/videos/vid-1" {
		t.Errorf("path = %s, want /videos/vid-1", gotPath)
	}
	if gotBody["upload_status"] != "COMPLETED" {
		t.Errorf("upload_status = %q, want COMPLETED", gotBody["upload_status"])
	}
}

func TestMarkUploadFailedSendsFailedStatus(t *testing.T) {
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
	}))
	defer srv.Close()

	if err := NewClient(srv.URL).MarkUploadFailed(context.Background(), "vid-2"); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	if gotBody["upload_status"] != "FAILED" {
		t.Errorf("upload_status = %q, want FAILED", gotBody["upload_status"])
	}
}

func TestMarkUploadCompleteSurfacesUpstreamFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	// The caller treats this as non-fatal, but it must still be reported so the
	// failure can be logged rather than silently swallowed here.
	if err := NewClient(srv.URL).MarkUploadComplete(context.Background(), "missing"); err == nil {
		t.Fatal("expected an error for a non-2xx response")
	}
}

func TestMarkUploadCompleteRequiresVideoID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("no request should be sent without a video id")
	}))
	defer srv.Close()

	if err := NewClient(srv.URL).MarkUploadComplete(context.Background(), ""); err == nil {
		t.Fatal("expected an error for an empty video id")
	}
}

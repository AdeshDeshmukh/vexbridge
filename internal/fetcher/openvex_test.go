package fetcher_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AdeshDeshmukh/vexbridge/internal/fetcher"
)

func TestOpenVEXFetcher_Fetch(t *testing.T) {
	sampleOpenVEX := `{
		"@context": "https://openvex.dev/ns/v1",
		"@id": "https://packages.cgr.dev/chainguard/vex/chainguard.openvex.json",
		"statements": [
			{
				"vulnerability": "CVE-2024-1234",
				"status": "not_affected",
				"justification": "component_not_present",
				"products": ["cgr.dev/chainguard/nginx:latest"]
			}
		]
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(sampleOpenVEX))
	}))
	defer ts.Close()

	f := fetcher.NewOpenVEXFetcher()
	stmts, digest, err := f.Fetch(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("OpenVEXFetch failed: %v", err)
	}

	if digest == "" {
		t.Error("expected non-empty SHA-256 digest")
	}

	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(stmts))
	}

	st := stmts[0]
	if st.VulnID != "CVE-2024-1234" {
		t.Errorf("got VulnID %q, want %q", st.VulnID, "CVE-2024-1234")
	}
	if st.Status != "not_affected" {
		t.Errorf("got Status %q, want %q", st.Status, "not_affected")
	}
	if st.Justification != "component_not_present" {
		t.Errorf("got Justification %q, want %q", st.Justification, "component_not_present")
	}
	if len(st.Products) != 1 || st.Products[0] != "cgr.dev/chainguard/nginx:latest" {
		t.Errorf("got Products %v, want %v", st.Products, []string{"cgr.dev/chainguard/nginx:latest"})
	}
}

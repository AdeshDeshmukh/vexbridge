package fetcher_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AdeshDeshmukh/vexbridge/internal/fetcher"
)

func TestCSAFFetcher_Fetch(t *testing.T) {
	sampleCSAF := `{
		"document": {
			"title": "Red Hat Security Advisory",
			"tracking": {
				"id": "RHSA-2024:1234"
			}
		},
		"vulnerabilities": [
			{
				"cve": "CVE-2024-5678",
				"product_status": {
					"known_not_affected": [
						"registry.access.redhat.com/ubi9/ubi:latest"
					],
					"fixed": [
						"registry.access.redhat.com/ubi8/ubi:8.9"
					]
				}
			}
		]
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(sampleCSAF))
	}))
	defer ts.Close()

	f := fetcher.NewCSAFFetcher()
	stmts, digest, err := f.Fetch(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("CSAFFetch failed: %v", err)
	}

	if digest == "" {
		t.Error("expected non-empty digest")
	}

	if len(stmts) != 2 {
		t.Fatalf("expected 2 statements (1 not_affected, 1 fixed), got %d", len(stmts))
	}

	st1 := stmts[0]
	if st1.VulnID != "CVE-2024-5678" || st1.Status != "not_affected" {
		t.Errorf("unexpected statement 0: %+v", st1)
	}
	if st1.Products[0] != "registry.access.redhat.com/ubi9/ubi:latest" {
		t.Errorf("unexpected product 0: %s", st1.Products[0])
	}

	st2 := stmts[1]
	if st2.VulnID != "CVE-2024-5678" || st2.Status != "fixed" {
		t.Errorf("unexpected statement 1: %+v", st2)
	}
	if st2.Products[0] != "registry.access.redhat.com/ubi8/ubi:8.9" {
		t.Errorf("unexpected product 1: %s", st2.Products[0])
	}
}

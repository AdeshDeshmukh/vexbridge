package joiner_test

import (
	"testing"

	"github.com/AdeshDeshmukh/vexbridge/internal/fetcher"
	"github.com/AdeshDeshmukh/vexbridge/internal/joiner"
)

func TestJoiner_SuppressesNotAffected(t *testing.T) {
	stmt := fetcher.Statement{
		VulnID:         "CVE-2024-5678",
		Status:         "not_affected",
		Products:       []string{"nginx:1.25"},
		SourceDocument: "https://test.example.com",
		StatementID:    "test-stmt-1",
	}
	j := joiner.New(func(vulnID, product string) (fetcher.Statement, bool) {
		if vulnID == "CVE-2024-5678" && product == "nginx:1.25" {
			return stmt, true
		}
		return fetcher.Statement{}, false
	})

	findings := []joiner.Finding{
		{VulnID: "CVE-2024-5678", ImageRef: "nginx:1.25", Severity: "HIGH"},
		{VulnID: "CVE-2024-9999", ImageRef: "nginx:1.25", Severity: "CRITICAL"},
	}

	results := j.Apply(findings)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if !results[0].Suppressed {
		t.Error("CVE-2024-5678 should be suppressed by vendor not_affected statement")
	}
	if results[0].SuppressedBy == nil || results[0].SuppressedBy.StatementID != "test-stmt-1" {
		t.Error("provenance not recorded correctly")
	}
	if results[1].Suppressed {
		t.Error("CVE-2024-9999 should not be suppressed — no matching statement")
	}
}

func TestJoiner_SuppressesFixed(t *testing.T) {
	stmt := fetcher.Statement{
		VulnID:         "CVE-2024-1111",
		Status:         "fixed",
		Products:       []string{"alpine:3.19"},
		SourceDocument: "https://test.example.com",
		StatementID:    "test-stmt-fixed",
	}
	j := joiner.New(func(vulnID, product string) (fetcher.Statement, bool) {
		if vulnID == "CVE-2024-1111" && product == "alpine:3.19" {
			return stmt, true
		}
		return fetcher.Statement{}, false
	})

	findings := []joiner.Finding{
		{VulnID: "CVE-2024-1111", ImageRef: "alpine:3.19", Severity: "MEDIUM"},
	}

	results := j.Apply(findings)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Suppressed {
		t.Error("CVE-2024-1111 should be suppressed by vendor fixed statement")
	}
	if results[0].SuppressedBy == nil || results[0].SuppressedBy.StatementID != "test-stmt-fixed" {
		t.Error("provenance not recorded correctly for fixed statement")
	}
}

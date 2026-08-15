package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
)

// csafDocument mirrors the structure of a CSAF VEX profile document.
type csafDocument struct {
	Document struct {
		Title    string `json:"title"`
		Tracking struct {
			ID string `json:"id"`
		} `json:"tracking"`
	} `json:"document"`
	Vulnerabilities []csafVuln `json:"vulnerabilities"`
}

type csafVuln struct {
	CVE             string         `json:"cve"`
	ProductStatuses csafProdStatus `json:"product_status"`
}

type csafProdStatus struct {
	KnownNotAffected []string `json:"known_not_affected"`
	Fixed            []string `json:"fixed"`
	KnownAffected    []string `json:"known_affected"`
}

// CSAFFetcher fetches and parses CSAF/VEX feeds (e.g., Red Hat VEX feeds).
type CSAFFetcher struct {
	http *HTTPClient
}

// NewCSAFFetcher constructs a CSAFFetcher.
func NewCSAFFetcher() *CSAFFetcher {
	return &CSAFFetcher{http: NewHTTPClient()}
}

func (f *CSAFFetcher) Fetch(ctx context.Context, url string) ([]Statement, string, error) {
	body, digest, err := f.http.Get(ctx, url)
	if err != nil {
		return nil, "", err
	}
	if body == nil {
		return nil, "", nil
	}

	var doc csafDocument
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, "", fmt.Errorf("parsing CSAF JSON: %w", err)
	}

	docID := doc.Document.Tracking.ID
	var stmts []Statement

	for _, v := range doc.Vulnerabilities {
		for _, prod := range v.ProductStatuses.KnownNotAffected {
			stmts = append(stmts, Statement{
				VulnID:         v.CVE,
				Status:         "not_affected",
				Products:       []string{prod},
				SourceDocument: url,
				StatementID:    fmt.Sprintf("%s/%s/not_affected/%s", docID, v.CVE, prod),
			})
		}
		for _, prod := range v.ProductStatuses.Fixed {
			stmts = append(stmts, Statement{
				VulnID:         v.CVE,
				Status:         "fixed",
				Products:       []string{prod},
				SourceDocument: url,
				StatementID:    fmt.Sprintf("%s/%s/fixed/%s", docID, v.CVE, prod),
			})
		}
	}
	return stmts, digest, nil
}

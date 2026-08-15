package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
)

// openVEXDocument mirrors the JSON-LD structure of an OpenVEX document.
type openVEXDocument struct {
	Context    string        `json:"@context"`
	ID         string        `json:"@id"`
	Statements []openVEXStmt `json:"statements"`
}

type openVEXStmt struct {
	VulnID        string   `json:"vulnerability"`
	Status        string   `json:"status"`
	Justification string   `json:"justification,omitempty"`
	Products      []string `json:"products"`
}

// OpenVEXFetcher fetches and parses OpenVEX JSON-LD feeds.
type OpenVEXFetcher struct {
	http *HTTPClient
}

// NewOpenVEXFetcher constructs an OpenVEXFetcher.
func NewOpenVEXFetcher() *OpenVEXFetcher {
	return &OpenVEXFetcher{http: NewHTTPClient()}
}

func (f *OpenVEXFetcher) Fetch(ctx context.Context, url string) ([]Statement, string, error) {
	body, digest, err := f.http.Get(ctx, url)
	if err != nil {
		return nil, "", err
	}
	if body == nil {
		return nil, "", nil
	}

	var doc openVEXDocument
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, "", fmt.Errorf("parsing OpenVEX JSON: %w", err)
	}

	stmts := make([]Statement, 0, len(doc.Statements))
	for i, s := range doc.Statements {
		stmts = append(stmts, Statement{
			VulnID:         s.VulnID,
			Status:         s.Status,
			Justification:  s.Justification,
			Products:       s.Products,
			SourceDocument: url,
			StatementID:    fmt.Sprintf("%s#stmt-%d", doc.ID, i),
		})
	}
	return stmts, digest, nil
}

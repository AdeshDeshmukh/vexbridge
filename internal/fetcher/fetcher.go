package fetcher

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Statement is the normalized, format-agnostic representation of a
// single VEX assertion that the joiner understands.
type Statement struct {
	// VulnID is the CVE or GHSA identifier (e.g. "CVE-2024-1234").
	VulnID string
	// Status is "not_affected", "fixed", "affected", or "under_investigation".
	Status string
	// Justification is the machine-readable reason (OpenVEX vocabulary).
	Justification string
	// Products are the image references this statement applies to.
	Products []string
	// SourceDocument is the URL of the feed this came from.
	SourceDocument string
	// StatementID uniquely identifies this statement within its document.
	StatementID string
}

// Fetcher fetches and parses a remote VEX feed into Statements.
type Fetcher interface {
	Fetch(ctx context.Context, url string) ([]Statement, string, error)
}

// HTTPClient wraps net/http with ETag-based conditional requests.
type HTTPClient struct {
	client  *http.Client
	mu      sync.Mutex
	lastTag map[string]string
}

// NewHTTPClient returns an HTTPClient with a 30-second timeout.
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client:  &http.Client{Timeout: 30 * time.Second},
		lastTag: make(map[string]string),
	}
}

// Get fetches url. Returns body bytes, SHA-256 hex digest, and any error.
// Returns nil body (no error) when the server responds 304 Not Modified.
func (h *HTTPClient) Get(ctx context.Context, url string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("building request: %w", err)
	}

	h.mu.Lock()
	if tag, ok := h.lastTag[url]; ok {
		req.Header.Set("If-None-Match", tag)
	}
	h.mu.Unlock()

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return nil, "", nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}

	h.mu.Lock()
	if etag := resp.Header.Get("ETag"); etag != "" {
		h.lastTag[url] = etag
	}
	h.mu.Unlock()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("reading body: %w", err)
	}

	digest := fmt.Sprintf("%x", sha256.Sum256(body))
	return body, digest, nil
}

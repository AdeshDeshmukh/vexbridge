package fetcher_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AdeshDeshmukh/vexbridge/internal/fetcher"
)

func TestHTTPClient_Get_ETagCaching(t *testing.T) {
	const bodyContent = `{"@context": "https://openvex.dev/ns/v1", "statements": []}`
	expectedDigest := fmt.Sprintf("%x", sha256.Sum256([]byte(bodyContent)))
	const etagValue = `"123456789"`

	requestCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.Header().Set("ETag", etagValue)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(bodyContent))
			return
		}
		if requestCount == 2 {
			if r.Header.Get("If-None-Match") != etagValue {
				t.Errorf("expected If-None-Match header %q, got %q", etagValue, r.Header.Get("If-None-Match"))
			}
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}))
	defer ts.Close()

	client := fetcher.NewHTTPClient()
	ctx := context.Background()

	// First fetch: expecting 200 OK + body + digest
	body, digest, err := client.Get(ctx, ts.URL)
	if err != nil {
		t.Fatalf("first Get failed: %v", err)
	}
	if string(body) != bodyContent {
		t.Errorf("got body %q, want %q", string(body), bodyContent)
	}
	if digest != expectedDigest {
		t.Errorf("got digest %q, want %q", digest, expectedDigest)
	}

	// Second fetch: expecting 304 Not Modified -> nil body
	body2, digest2, err2 := client.Get(ctx, ts.URL)
	if err2 != nil {
		t.Fatalf("second Get failed: %v", err2)
	}
	if body2 != nil {
		t.Errorf("expected nil body for 304 Not Modified, got %q", string(body2))
	}
	if digest2 != "" {
		t.Errorf("expected empty digest for 304 Not Modified, got %q", digest2)
	}
}

func TestHTTPClient_Get_HTTPErrorStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := fetcher.NewHTTPClient()
	_, _, err := client.Get(context.Background(), ts.URL)
	if err == nil {
		t.Fatal("expected error for 500 status code, got nil")
	}
}

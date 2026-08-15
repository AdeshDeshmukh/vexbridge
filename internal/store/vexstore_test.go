package store_test

import (
	"sync"
	"testing"

	"github.com/AdeshDeshmukh/vexbridge/internal/fetcher"
	"github.com/AdeshDeshmukh/vexbridge/internal/store"
)

func TestVEXStore_NoStateLeakBetweenSources(t *testing.T) {
	s := store.New()

	feedA := []fetcher.Statement{{
		VulnID:         "CVE-2024-1111",
		Status:         "not_affected",
		Products:       []string{"nginx:1.25"},
		SourceDocument: "https://feed-a.example.com",
	}}
	feedB := []fetcher.Statement{{
		VulnID:         "CVE-2024-2222",
		Status:         "fixed",
		Products:       []string{"alpine:3.19"},
		SourceDocument: "https://feed-b.example.com",
	}}

	s.Upsert(feedA)
	s.Upsert(feedB)

	if _, ok := s.Lookup("CVE-2024-1111", "nginx:1.25"); !ok {
		t.Fatal("expected CVE-2024-1111 statement from feed A")
	}

	s.Reset()

	if _, ok := s.Lookup("CVE-2024-1111", "nginx:1.25"); ok {
		t.Fatal("state from feed A leaked across Reset()")
	}
	if _, ok := s.Lookup("CVE-2024-2222", "alpine:3.19"); ok {
		t.Fatal("state from feed B leaked across Reset()")
	}
}

func TestVEXStore_ConcurrentUpsert(t *testing.T) {
	s := store.New()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			s.Upsert([]fetcher.Statement{{
				VulnID:   "CVE-2024-9999",
				Status:   "not_affected",
				Products: []string{"image:latest"},
			}})
		}(i)
	}
	wg.Wait()

	if _, ok := s.Lookup("CVE-2024-9999", "image:latest"); !ok {
		t.Fatal("concurrent upsert lost the statement")
	}
}

func TestVEXStore_Snapshot(t *testing.T) {
	s := store.New()
	s.Upsert([]fetcher.Statement{{
		VulnID:   "CVE-2024-3333",
		Status:   "fixed",
		Products: []string{"python:3.11"},
	}})

	snap := s.Snapshot()
	if len(snap) != 1 {
		t.Fatalf("expected 1 snapshot statement, got %d", len(snap))
	}
	if snap[0].VulnID != "CVE-2024-3333" {
		t.Errorf("got snapshot VulnID %s, want CVE-2024-3333", snap[0].VulnID)
	}
}

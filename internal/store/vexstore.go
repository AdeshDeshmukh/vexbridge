package store

import (
	"sync"

	"github.com/AdeshDeshmukh/vexbridge/internal/fetcher"
)

// Key uniquely identifies a (vulnerability, product) pair.
type Key struct {
	VulnID  string
	Product string
}

// VEXStore holds the current set of vendor VEX statements in memory,
// deduplicated by (VulnID, Product) with last-writer-wins semantics.
type VEXStore struct {
	mu    sync.RWMutex
	index map[Key]fetcher.Statement
}

// New returns an empty VEXStore.
func New() *VEXStore {
	return &VEXStore{index: make(map[Key]fetcher.Statement)}
}

// Upsert adds or replaces statements. If two feeds publish different
// statuses for the same (vuln, product), the most recent call wins.
func (s *VEXStore) Upsert(stmts []fetcher.Statement) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, st := range stmts {
		for _, prod := range st.Products {
			s.index[Key{VulnID: st.VulnID, Product: prod}] = st
		}
	}
}

// Lookup returns the statement for a (vulnID, product) pair if one exists.
func (s *VEXStore) Lookup(vulnID, product string) (fetcher.Statement, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st, ok := s.index[Key{VulnID: vulnID, Product: product}]
	return st, ok
}

// Reset clears all statements. Used between reconcile cycles for a
// specific VEXSource so stale entries from a removed feed do not linger.
func (s *VEXStore) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.index = make(map[Key]fetcher.Statement)
}

// Snapshot returns a copy of all current statements (for persistence).
func (s *VEXStore) Snapshot() []fetcher.Statement {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]fetcher.Statement, 0, len(s.index))
	for _, v := range s.index {
		out = append(out, v)
	}
	return out
}

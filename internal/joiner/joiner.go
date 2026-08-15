package joiner

import "github.com/AdeshDeshmukh/vexbridge/internal/fetcher"

// Finding is one CVE finding from a Grype scan.
type Finding struct {
	VulnID   string
	ImageRef string
	Severity string
	PkgName  string
}

// Result is the disposition of a Finding after VEX join.
type Result struct {
	Finding    Finding
	Suppressed bool
	// SuppressedBy is the VEX statement that caused suppression, if any.
	SuppressedBy *fetcher.Statement
}

// Joiner applies vendor VEX statements to raw Grype findings.
type Joiner struct {
	lookup func(vulnID, product string) (fetcher.Statement, bool)
}

// New returns a Joiner backed by the provided store lookup function.
func New(lookup func(vulnID, product string) (fetcher.Statement, bool)) *Joiner {
	return &Joiner{lookup: lookup}
}

// Apply returns a Result for every input Finding. Findings matched by
// a "not_affected" or "fixed" statement are marked Suppressed with
// the responsible statement recorded for provenance.
func (j *Joiner) Apply(findings []Finding) []Result {
	results := make([]Result, 0, len(findings))
	for _, f := range findings {
		stmt, ok := j.lookup(f.VulnID, f.ImageRef)
		if ok && (stmt.Status == "not_affected" || stmt.Status == "fixed") {
			s := stmt
			results = append(results, Result{
				Finding:      f,
				Suppressed:   true,
				SuppressedBy: &s,
			})
			continue
		}
		results = append(results, Result{Finding: f})
	}
	return results
}

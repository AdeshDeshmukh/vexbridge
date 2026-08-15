# VEXBridge — Design Document

> **Status:** Active  
> **Author:** Adesh Kishor Deshmukh ([@AdeshDeshmukh](https://github.com/AdeshDeshmukh))  
> **Last updated:** August 2026

---

## Overview

This document records the architectural decisions behind VEXBridge, the reasoning behind each choice, and the trade-offs accepted. It is intended for contributors, reviewers, and anyone evaluating whether VEXBridge's approach is sound before building on it.

VEXBridge solves one scoped problem: Kubescape's `kubevuln` generates VEX documents from its own scan results but does not consume VEX published by external vendors. This leaves users seeing CVE findings that upstream vendors (Red Hat, Chainguard) have already marked `not_affected` or `fixed`. VEXBridge ingests those feeds and suppresses the corresponding findings before they reach the user.

---

## Problem Decomposition

The upstream issue ([kubescape/kubevuln#387](https://github.com/kubescape/kubevuln/issues/387)) identifies four missing pieces:

| Gap | VEXBridge Component |
|-----|---------------------|
| No way to declare an external feed | `VEXSource` CRD |
| No periodic fetch of feed documents | `VEXSourceReconciler` + `Fetcher` |
| No normalization across formats | `OpenVEXFetcher`, `CSAFFetcher` → common `Statement` model |
| No join step to suppress matching findings | `Joiner` |

Each component is decoupled and independently testable. The join step, in particular, does not depend on the controller being healthy — it reads from the `VEXStore` which survives reconcile failures.

---

## Component Design

### VEXSource CRD

**Why a CRD instead of a ConfigMap or Helm values?**

Feed declarations need:
- **Structured validation** — `format` must be one of `OpenVEX` or `CSAF`. A ConfigMap value is an unvalidated string.
- **Status subresource** — `lastSyncTime`, `feedHash`, and `statementCount` need to be updated by the controller without triggering reconcile loops on the spec. The status subresource split makes this correct.
- **RBAC scoping** — cluster operators should be able to grant teams permission to create `VEXSource` objects in their namespace without granting access to cluster-wide config. ConfigMap RBAC is namespace-scoped too, but the CRD approach makes this explicit and auditable.
- **`kubectl get` discoverability** — `kubectl get vexsource -A` gives an operator an instant inventory of all declared feeds and their sync health. No equivalent exists for ConfigMaps.

The CRD lives at `security.vexbridge.io/v1alpha1`. The `v1alpha1` version signals that the spec shape may change during the LFX term; a `v1beta1` graduation path is reserved for when the API stabilises.

---

### Fetcher Interface and Format Parsers

```go
type Fetcher interface {
    Fetch(ctx context.Context, url string) ([]Statement, string, error)
}
```

The interface returns `([]Statement, digest, error)`. The `digest` is a SHA-256 hex string of the response body. The controller compares this against `status.feedHash` before writing to the store — a feed that hasn't changed since the last cycle produces no write and no reconcile noise.

**ETag-based conditional requests** are sent on every cycle. If the server returns `304 Not Modified`, `Fetch` returns `(nil, "", nil)`. The controller treats a `nil` body as "no change" and skips the upsert. This matters for Red Hat's feed, which is a large directory listing — avoiding a full re-parse on every cycle is not a micro-optimization.

**Why two separate parsers instead of one unified parser?**

OpenVEX (JSON-LD) and CSAF 2.0 have fundamentally different shapes:

- OpenVEX has a flat `statements[]` array where each statement directly names affected products.
- CSAF uses a hierarchical `vulnerabilities[].product_status` object with product trees that require lookup by product ID.

A unified parser would need to branch on format type internally, producing the same two code paths but with worse readability and harder-to-test failure modes. Two concrete `Fetcher` implementations are injected into the reconciler via a `map[VEXFormat]Fetcher`, making it straightforward to add a third format (e.g., CycloneDX VEX) without modifying existing parsers.

---

### VEXStore — Deduplication Model

The store is a `sync.RWMutex`-protected `map[Key]Statement` where `Key = (VulnID, Product)`.

**Why last-writer-wins on conflicts?**

When two feeds publish different statuses for the same `(CVE, imageRef)` pair, the most recent `Upsert` call wins. The alternatives considered:

| Strategy | Problem |
|----------|---------|
| Reject the second statement | Requires a conflict resolution API; users must understand it |
| Merge (most permissive wins) | Hides genuine vendor disagreements; `not_affected` from one feed silences `affected` from another |
| Merge (most conservative wins) | `affected` always beats `not_affected` — defeats the purpose of VEX |
| Last-writer-wins | Simple, predictable, auditable via `status.feedHash` |

Last-writer-wins is correct because feed refresh order is stable within a reconcile cycle (feeds are processed in the order they appear in the VEXSource list). Operators who want a specific feed to take precedence can give it a lower `refreshInterval`.

**State isolation between reconcile cycles.**

The store's `Reset()` method is called at the start of each full reconcile sweep for a given `VEXSource`. This prevents stale statements from a removed feed persisting indefinitely. The `TestVEXStore_NoStateLeakBetweenSources` test verifies this contract.

The singleton-state problem described in kubevuln#387 — where a shared mutable store accumulates statements from deleted feeds — is prevented by this `Reset()`-per-source pattern rather than by trying to track which statements came from which feed (which would require a second index and complicate the dedup logic).

---

### Joiner — Suppression Semantics

The Joiner matches a Grype finding to a VEX statement by `(VulnID, imageRef)`. A finding is suppressed if and only if the matched statement has `status = "not_affected"` or `status = "fixed"`. Statements with `status = "affected"` or `status = "under_investigation"` do not suppress.

**Provenance is always recorded.** Every suppressed finding carries a `SuppressedBy` field pointing to the `StatementID` and `SourceDocument` of the vendor statement that caused the suppression. This is non-negotiable: a tool that silently hides CVEs with no audit trail is worse than a tool that shows too many.

**Why not suppress `under_investigation`?**

An `under_investigation` status means the vendor has not yet determined whether their product is affected. Suppressing on this basis would hide genuine unknowns. The proposal's own language — "suppress `not_affected` / `fixed` statements" — is precise; this implementation follows it exactly.

---

### Relation to SecurityException CRD

The LFX proposal notes that vendor `not_affected` statements are "functionally the same shape as a user-authored exception." VEXBridge's `Statement` model is deliberately designed to be compatible with the SecurityException matching model: both are keyed on `(VulnID, imageRef)` and carry a justification string.

This means the Joiner can be extended in a follow-up to handle both `VEXSource`-derived suppressions and `SecurityException`-derived suppressions through the same code path, with provenance distinguishing which kind caused each suppression.

VEXBridge does not implement SecurityException itself — that is a separate CRD tracked separately in the Kubescape roadmap. The interface boundary is kept clean.

---

## Where FleetReport Should Live (OSS vs Platform Boundary)

The proposal leaves open the question of where aggregated VEX results — a kind of "FleetReport" — should live. The recommendation here:

**VEXBridge stays read-only ingestion. Aggregated reporting belongs in a separate CRD or the Kubescape UI, not in VEXBridge.**

Reasons:
1. Aggregation semantics depend on how findings are grouped (by namespace, by workload, by image family) — a policy question that varies per cluster operator.
2. A `FleetReport` CRD inside VEXBridge would couple the ingestion controller to a reporting model, making both harder to evolve independently.
3. The `OpenVulnerabilityExchangeContainer` objects VEXBridge writes are already accessible to anything that reads the Kubernetes API — a separate reporting layer can query them without VEXBridge needing to know it exists.

---

## Operational Characteristics

### Controller Restart Behavior

On restart, the reconciler re-fetches all declared feeds. ETag caching means most feeds return `304 Not Modified`, so the restart cost is proportional to the number of feeds that actually changed since the last sync, not the total feed volume.

The `VEXStore` is in-memory and does not survive restarts. This is intentional — re-populating from the feeds on startup ensures the store is always consistent with the current feed state, even if the previous run crashed mid-upsert.

### Scaling Considerations

Each `VEXSource` object is reconciled independently. A cluster with 20 declared feeds will run 20 independent HTTP fetches on each refresh cycle. The `refreshInterval` field allows operators to stagger fetches by feed, reducing peak egress.

The `VEXStore` is a single shared instance accessed under `sync.RWMutex`. Read contention from the Joiner side (many concurrent Grype results being joined simultaneously) is bounded by the number of active scan goroutines in `kubevuln`, which is already rate-limited by `kubevuln`'s `CooldownQueue`.

---

## Explicit Non-Goals

| Out of Scope | Reason |
|---|---|
| Authoring or editing VEX inside the cluster | The proposal explicitly excludes this; it is a separate surface (Kubescape UI / ARMO platform) |
| Vulnerability database mirroring | VEXBridge fetches vendor *assessments*, not the underlying CVE data itself |
| Supporting every CSAF profile | Only the VEX profile (`csaf_vex`) is parsed. CSAF advisories (`csaf_base`, `csaf_security_advisory`) are ignored |
| Multi-document conflict resolution DSL | Last-writer-wins is sufficient for the current scope; a DSL can be introduced in a `v1beta1` revision |
| Grype version pinning | VEXBridge does not depend on a specific Grype version; the Joiner operates on parsed output, not Grype internals |

---

## Future Work

- **`v1beta1` API** — graduate the CRD once the spec has been validated against real production feeds. Likely changes: add a `credentialsRef` field for authenticated feeds, add a `tlsConfig` field for private registries.
- **Native `go-vex` library integration** — the current OpenVEX parser is a minimal JSON-LD decoder. Replacing it with `github.com/openvex/go-vex` gives full spec compliance at the cost of an additional dependency.
- **Metrics** — expose `vexbridge_statements_ingested_total{source, format}` and `vexbridge_fetch_duration_seconds{source}` as Prometheus metrics via controller-runtime's built-in metrics server.
- **Webhook validation** — reject `VEXSource` objects with malformed URLs or unknown format values at admission time rather than at first reconcile.

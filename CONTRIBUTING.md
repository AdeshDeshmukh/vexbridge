# Contributing to VEXBridge

Thank you for taking the time to contribute. VEXBridge is a focused project — external VEX ingestion for Kubescape — so contributions that stay within that scope have the highest chance of being accepted quickly.

This document covers local setup, the PR workflow, and what the maintainers look for in a review.

---

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Before You Start](#before-you-start)
- [Local Development Setup](#local-development-setup)
- [Project Layout](#project-layout)
- [Making Changes](#making-changes)
- [Testing](#testing)
- [Pull Request Checklist](#pull-request-checklist)
- [Commit Message Format](#commit-message-format)
- [What We Look For in Review](#what-we-look-for-in-review)

---

## Code of Conduct

VEXBridge follows the [CNCF Community Code of Conduct](https://github.com/cncf/foundation/blob/main/code-of-conduct.md). Be direct, be constructive, and assume good faith.

---

## Before You Start

- For **bug fixes** and **small improvements**: open a PR directly. No issue needed.
- For **new features** or **API changes** (new fields on `VEXSourceSpec`, new format support): open an issue first and describe what you want to build. This avoids duplicate work and keeps changes aligned with the DESIGN.md rationale.
- For **questions about the design**: open a GitHub Discussion, not an issue.

---

## Local Development Setup

### Prerequisites

```bash
go version      # 1.22+
kubectl version # 1.28+
kind version    # 0.23+
docker version  # any
golangci-lint --version  # 1.59+
```

### Clone and initialize

```bash
git clone https://github.com/AdeshDeshmukh/vexbridge.git
cd vexbridge
go mod download
```

### Install the CRD on a local cluster

```bash
kind create cluster --name vexbridge-dev
kubectl apply -f config/crd/vexsource.yaml
```

### Run the controller locally (outside the cluster)

```bash
go run ./cmd/vexbridge \
  --health-probe-bind-address=:8081 \
  --metrics-bind-address=:8080 \
  --leader-elect=false
```

The controller will connect to whatever cluster your current `KUBECONFIG` points to. On a kind cluster this works out of the box.

### Apply a sample feed

```bash
kubectl apply -f config/samples/redhat-vex-source.yaml

# Watch sync status
kubectl get vexsource -w -n vexbridge-system
```

---

## Project Layout

```
internal/
├── controller/     # Reconciler — owns the sync lifecycle
├── fetcher/        # HTTP client + per-format parsers
├── store/          # In-memory VEX statement index
├── joiner/         # Suppression logic — match findings against statements
└── grype/          # Grype CLI wrapper
```

Each package has a single, clear responsibility. If a change touches more than two packages, the design should be revisited.

---

## Making Changes

### Branch naming

```
feat/short-description       # new functionality
fix/short-description        # bug fix
refactor/short-description   # internal change, no behaviour change
docs/short-description       # documentation only
test/short-description       # tests only
```

### Regenerating CRD manifests

If you change types in `api/v1alpha1/`, regenerate the CRD manifest:

```bash
make generate
# controller-gen object paths="./api/..." output:dir=./api/v1alpha1
# controller-gen crd paths="./api/..." output:dir=./config/crd
```

Commit both the updated types and the updated manifest in the same commit.

### Adding a new VEX format

1. Create `internal/fetcher/<format>.go` implementing the `Fetcher` interface.
2. Add the new format constant to `api/v1alpha1/vexsource_types.go`.
3. Register it in `cmd/vexbridge/main.go`'s `Fetchers` map.
4. Add a fixture file to `test/fixtures/`.
5. Add a unit test in `internal/fetcher/<format>_test.go`.
6. Update the "Supported Feeds" table in `README.md`.

---

## Testing

### Unit tests

```bash
make test
# go test ./... -race -count=1
```

All packages must maintain `race`-clean tests. The `-count=1` flag disables the test cache — results should be deterministic and not reliant on cached state.

### End-to-end test (real Red Hat CSAF feed)

```bash
kind create cluster --name vexbridge-e2e
kubectl apply -f config/crd/vexsource.yaml
go run ./cmd/vexbridge &

make e2e
# go test -v -tags=e2e ./test/e2e/... -timeout 5m
```

The e2e test applies a real `VEXSource` pointing to Red Hat's CSAF feed and waits up to two minutes for `status.statementCount > 0`. It requires outbound internet access from the test runner.

### Lint

```bash
make lint
# golangci-lint run ./...
```

The lint config is in `.golangci.yaml`. Key rules: `errcheck`, `govet`, `staticcheck`, `unused`. PRs with lint failures will not be merged.

---

## Pull Request Checklist

Before marking your PR ready for review, confirm:

- [ ] `make test` passes with `-race`
- [ ] `make lint` passes with no warnings
- [ ] New packages have at least one test
- [ ] `VEXSourceSpec` changes are reflected in the generated CRD manifest (`make generate`)
- [ ] Public Go symbols that are part of the API have godoc comments
- [ ] `README.md` is updated if you added a new format or changed the CRD spec
- [ ] Commit messages follow the format below

---

## Commit Message Format

VEXBridge uses [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <short summary>

[optional body]

[optional footer]
```

**Types:** `feat`, `fix`, `refactor`, `test`, `docs`, `build`, `chore`  
**Scopes:** `controller`, `fetcher`, `store`, `joiner`, `api`, `cmd`, `e2e`, `docs`

**Examples:**

```
feat(fetcher): add CycloneDX VEX format parser

fix(store): prevent stale statements after VEXSource deletion

test(store): add concurrent upsert race test

docs(design): document last-writer-wins dedup rationale
```

Squash commits before merging. The PR title becomes the merge commit message, so make it a valid Conventional Commit line.

---

## What We Look For in Review

### Correctness first

- Does the change handle the error path? Every `err` must be checked.
- Does the change preserve thread safety? `VEXStore` is accessed concurrently — any modification to it requires a lock analysis.
- Does the change preserve state isolation? `Reset()` must be called before re-populating a source's statements. See `TestVEXStore_NoStateLeakBetweenSources`.

### API stability

`VEXSourceSpec` is `v1alpha1`. We can add optional fields freely. We cannot remove fields or change their types without a version bump. Additions should be `omitempty` with a documented default.

### Godoc comments

Exported symbols that are part of the controller's public surface — the `Fetcher` interface, `Statement` struct, `VEXStore` methods — must have godoc comments. Internal helpers do not require comments unless the name does not carry the meaning.

### Test coverage

A new code path without a test will be sent back for a test. The test does not need to be exhaustive — it needs to cover the behaviour the PR claims to add or fix.

---

## Questions

Open a [GitHub Discussion](https://github.com/AdeshDeshmukh/vexbridge/discussions) or reach out to the LFX mentors:
- Matthias Bertschy ([@matthyx](https://github.com/matthyx))
- Ben Hirschberg ([@slashben](https://github.com/slashben))

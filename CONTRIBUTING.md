# Contributing to VEXBridge

Thank you for your interest in contributing to VEXBridge!

## Prerequisites
- Go 1.22+
- Docker & `kind` (for E2E tests)
- `kubectl`

## Local Development & Testing

```bash
# Run unit tests with race detection
make test

# Build local binary
make build

# Build docker image
make docker-build
```

## Pull Request Guidelines
1. Ensure all code passes `make test`.
2. Format code according to standard Go conventions (`gofmt`).
3. Include unit tests for any new parser, store, or reconciler changes.

BINARY      := vexbridge
IMAGE       := ghcr.io/adeshdeshmukh/vexbridge
TAG         := $(shell git rev-parse --short HEAD 2>/dev/null || echo "latest")

.PHONY: build test lint generate docker-build deploy-crd e2e

build:
	go build -o bin/$(BINARY) ./cmd/vexbridge

test:
	go test ./... -race -count=1

lint:
	golangci-lint run ./...

generate:
	controller-gen object paths="./api/..." output:dir=./api/v1alpha1
	controller-gen crd paths="./api/..." output:dir=./config/crd

docker-build:
	docker build -t $(IMAGE):$(TAG) .

deploy-crd:
	kubectl apply -f config/crd/

e2e:
	go test -v -tags=e2e ./test/e2e/... -timeout 5m

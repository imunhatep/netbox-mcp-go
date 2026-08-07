BINARY := netbox-mcp
PKG := github.com/imunhatep/netbox-mcp-go
VERSION ?= dev
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
LDFLAGS := -X $(PKG)/internal/version.Version=$(VERSION) -X $(PKG)/internal/version.Commit=$(COMMIT)

IMAGE := netbox-mcp
PLATFORMS := linux/amd64,linux/arm64

.PHONY: build test tidy run image image-multiarch
build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/netbox-mcp
test:
	go test ./...
tidy:
	go mod tidy
run: build
	./bin/$(BINARY) serve

# Build a single-arch image for the local host.
image:
	podman build \
		--build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) \
		-t $(IMAGE):$(VERSION) .

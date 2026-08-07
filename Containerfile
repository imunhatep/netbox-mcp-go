# syntax=docker/dockerfile:1

# Multi-arch build: pass --platform (e.g. linux/amd64,linux/arm64) via buildx/podman.
# BuildKit injects BUILDPLATFORM/TARGETOS/TARGETARCH so the Go toolchain runs
# natively on the build host and cross-compiles for the target arch.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=none

WORKDIR /src

# Cache modules separately from source for faster rebuilds.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath \
    -ldflags "-s -w \
    -X github.com/imunhatep/netbox-mcp-go/internal/version.Version=${VERSION} \
    -X github.com/imunhatep/netbox-mcp-go/internal/version.Commit=${COMMIT}" \
    -o /out/netbox-mcp ./cmd/netbox-mcp

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/netbox-mcp /usr/local/bin/netbox-mcp

# Listen on all interfaces so the server is reachable from outside the container.
ENV NETBOX_MCP_LISTEN=0.0.0.0:3031
EXPOSE 3031

USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/netbox-mcp"]
CMD ["serve"]

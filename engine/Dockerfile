# path: engine/Dockerfile
#
# Container image for the dq-engine binary per ADR-0042 §"Clause 1
# — Docker invariants" and B2-28's slice. Builds the engine via a
# multi-stage build, copying only the resulting static binary into
# a minimal non-root runtime image.
#
# Posture:
#   - Multi-stage: builder produces a static linux binary; runtime
#     stage carries only the binary.
#   - Non-root: the runtime image's `nonroot` user (UID 65532)
#     matches the UID pinned by deploy/base/deployment.yaml.
#   - readOnlyRootFilesystem-friendly: the binary writes nothing to
#     the container filesystem at runtime (engine logs go to stderr;
#     state lives in BigQuery, GCS, and Pub/Sub per ADR-0010).
#   - Dropped capabilities + RuntimeDefault seccomp: enforced by the
#     pod's securityContext (deploy/base/deployment.yaml:91-108); the
#     image layer is compatible because the binary never invokes
#     setuid/setcap/raw-socket operations.
#   - Image name `dq-engine:<tag>` per ADR-0042 Clause 1: built by
#     the root Makefile's `build-engine-image` target which derives
#     <tag> from the git tag's `engine-v` prefix per the ADR-0042
#     Clause 3 derivation rule.
#
# Build context: the engine module directory (`./engine`). The
# engine module is self-contained (no imports under
# `dq-platform/tools/...`); the surrounding `go.work` is bypassed
# inside the container, which is the recommended pattern for
# go-workspace projects with self-contained module subgraphs.
#
# Cross-architecture: ARG TARGETARCH is honored so `docker
# buildx` can produce arm64 + amd64 manifests. Local single-arch
# builds use the daemon's native arch by default.

# --- Builder stage --------------------------------------------------
# Pinned to the Go toolchain version declared in engine/go.mod's
# `go 1.25.0` directive. Bumping engine/go.mod's directive without
# bumping this digest is a contract violation caught at the next
# image build (the toolchain refuses to compile modules whose
# go.mod specifies a newer toolchain).
FROM golang:1.25-bookworm AS builder

ARG TARGETARCH

WORKDIR /workspace

# Module-cache layer: copy go.mod/go.sum first so `go mod download`
# caches independently of source changes. Subsequent source edits
# don't bust this layer unless the dependency set changes.
COPY go.mod go.sum ./
RUN go mod download

# Now copy the rest of the engine module's source tree.
COPY . .

# Build a static linux binary. -trimpath removes the build host's
# absolute path from the binary (reproducibility); -ldflags="-s -w"
# strips symbol/debug info (smaller image).
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags="-s -w" \
    -o /out/dq-engine \
    ./cmd/dq-engine

# --- Runtime stage --------------------------------------------------
# Distroless static-debian12:nonroot is the minimum-surface runtime
# image for static Go binaries: no shell, no package manager, no
# coreutils; a single non-root user `nonroot` at UID 65532 (matching
# deploy/base/deployment.yaml:102 runAsUser); CA certificates
# included so the binary can perform HTTPS dials (BigQuery, GCS,
# Pub/Sub). The pinned UID + readOnlyRootFilesystem invariants from
# ADR-0042 Clause 1 are satisfied at the image layer.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/dq-engine /usr/local/bin/dq-engine

# The distroless `nonroot` user (UID 65532) is the image default
# and matches deploy/base/deployment.yaml's runAsUser pin.
USER nonroot:nonroot

ENTRYPOINT ["/usr/local/bin/dq-engine"]

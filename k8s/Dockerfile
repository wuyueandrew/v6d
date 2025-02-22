# Build the manager binary
FROM golang:1.19 as builder

ENV GO111MODULE=on
ENV CGO_ENABLED=0
ENV GOOS=linux

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY apis/ apis/
COPY controllers/ controllers/
COPY cmd/ cmd/
COPY pkg/ pkg/
COPY hack/ hack/

# Build
RUN go build -a -o vineyardctl cmd/main.go && \
    strip vineyardctl || true

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /

COPY config/scheduler/config.yaml /etc/kubernetes/scheduler.yaml

COPY --from=builder /workspace/vineyardctl /vineyardctl
USER nonroot:nonroot

ENTRYPOINT ["/vineyardctl", "manager"]

# Generate kubernetes resources
FROM ubuntu:20.04 as resgen
ARG IMG
ENV ENV TZ="Europe/Warsaw"
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get -y install ca-certificates uuid-runtime make golang-1.21-go
ENV PATH="$PATH:/usr/lib/go-1.21/bin"

RUN mkdir /workspace
COPY . /workspace
WORKDIR /workspace
RUN make generate-operator-resources IMG="$IMG" OUT_FILE="operator-resources.yaml"


# Build the manager binary
FROM golang:1.20 as builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/main.go cmd/main.go
COPY print/main.go print/main.go
COPY api/ api/
COPY internal/controller/ internal/controller/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager cmd/main.go
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o print-resources print/main.go


# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
COPY --from=builder /workspace/print-resources .
COPY --from=resgen /workspace/operator-resources.yaml /
USER 65532:65532

CMD ["/manager"]

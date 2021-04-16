# Build the openfunction binary
FROM golang:1.13 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN GOPROXY="https://goproxy.io" go mod download

# Copy the go source
COPY cmd/main.go main.go
COPY pkg/ pkg/

# Build
RUN GOPROXY="https://goproxy.io" CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o openfunction main.go

# Use distroless as minimal base image to package the openfunction binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM openfunction/distroless-static:nonroot
WORKDIR /
COPY --from=builder /workspace/openfunction .
USER nonroot:nonroot

ENTRYPOINT ["/openfunction"]

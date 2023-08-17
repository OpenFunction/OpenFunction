# Copyright 2022 The OpenFunction Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

ARG GOPROXY="https://goproxy.cn"

# Build the openfunction binary
FROM golang:1.19 as builder
ARG GOPROXY

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go env -w GOPROXY=${GOPROXY} && go mod download

# Copy the go source
COPY main.go main.go
COPY apis/ apis/
COPY controllers/ controllers/
COPY pkg/ pkg/

# Build
RUN GOPROXY=${GOPROXY} CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o openfunction main.go

# Use distroless as minimal base image to package the openfunction binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM openfunction/distroless-static:nonroot
WORKDIR /
COPY --from=builder /workspace/openfunction .
USER 65532:65532

ENTRYPOINT ["/openfunction"]

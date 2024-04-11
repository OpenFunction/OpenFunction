# OpenFunction + Wasmedge Integration

## Motivation

WasmEdge is a lightweight, high-performance, and extensible WebAssembly runtime for cloud native, edge, and decentralized applications. It powers serverless apps, embedded functions, microservices, smart contracts, and IoT devices.

The WasmEdge Runtime provides a well-defined execution sandbox for its contained WebAssembly bytecode program. The runtime offers isolation and protection for operating system resources and memory space. Adopting wasmedge as a new runtime for OpenFunction is powerful and valuable.

## Goals

- Supports building container images from webassembly source code
- Supports running wasm functions by configuring function CR

## Proposal

### Runtime

#### Install WasmEdge

Use the simple install script to install WasmEdge.

```
wget -qO- https://raw.githubusercontent.com/WasmEdge/WasmEdge/master/utils/install.sh | bash -s -- -p /usr/local
```

#### Build and install crun

The crun project has WasmEdge support baked in. For now, the easiest approach is just built it yourself from source. First, let's make sure that crun dependencies are installed on your Ubuntu 20.04.

```sehll
sudo apt update
sudo apt install -y make git gcc build-essential pkgconf libtool \
    libsystemd-dev libprotobuf-c-dev libcap-dev libseccomp-dev libyajl-dev \
    go-md2man libtool autoconf python3 automake
```

Next, configure, build, and install a crun binary with WasmEdge support.

```
git clone https://github.com/containers/crun
cd crun
./autogen.sh
./configure --with-wasmedge
make
sudo make install
```

#### Setup Containerd

Edit the configuration `/etc/containerd/config.toml`, add the following section to setup crun runtime, make sure the BinaryName equal to your crun binary path
```
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.crun]
    runtime_type = "io.containerd.runc.v2"
    pod_annotations = ["*.wasm.*", "wasm.*", "module.wasm.image/*", "*.module.wasm.image", "module.wasm.image/variant.*"]
    privileged_without_host_devices = false
    [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.crun.options]
      BinaryName = "/usr/local/bin/crun"
```

Restart containerd service ```sudo systemctl restart containerd```.

#### Add crun runtimeClass

```
$ cat > runtime.yaml <<EOF
apiVersion: node.k8s.io/v1
kind: RuntimeClass
metadata:
  name: openfunction-crun
handler: crun
EOF
$ kubectl apply runtime.yaml
```

### build

To support building container images from wasm source code，adjustments that need to be made:

- Add a new ClusterBuildStrategy named `wasmedge`，upgrade the version of `buildah` to `1.28.0` and configure the `build-args` parameter
- Upgrade the version of `shipwright-build` to `v0.10.0`
- User needs to specify `spec.build.dockerfile` field

When the value of the `spec.workloadRuntime` field is `wasmedge` or the annotations of the Function CR contains `module.wasm.image/variant: compat-smart`, `spec.build.shipwright.strategy` will be automatically generated based on the ClusterBuildStrategy named wasmedge.


```yaml=
apiVersion: core.openfunction.io/v1beta2
kind: Function
metadata:
  name: wasmedge-http-server-build
  # Optional
  # annotations:
  #   module.wasm.image/variant: compat-smart
spec:
  workloadRuntime: wasmedge
  image: openfunctiondev/wasmedge_http_server:0.1.0
  imageCredentials:
    name: push-secret
  build:
    # shipwright:
    # strategy will be automatically generated based on workloadRuntime
    #   strategy:
    #     name: wasmedge
    #     kind: ClusterBuildStrategy
    dockerfile: "Dockerfile"
    srcRepo:
      url: "https://github.com/OpenFunction/samples"
      sourceSubPath: "functions/knative/wasmedge/http-server"
      revision: "main"
  port: 8080
```

Here is an example `Dockfile`：
```dockerfile=
FROM --platform=$BUILDPLATFORM rust:1.64 AS buildbase
RUN rustup target add wasm32-wasi
WORKDIR /src

FROM --platform=$BUILDPLATFORM buildbase AS buildserver
COPY server/ /src
RUN --mount=type=cache,target=/usr/local/cargo/git/db \
    --mount=type=cache,target=/usr/local/cargo/registry/cache \
    --mount=type=cache,target=/usr/local/cargo/registry/index \
    cargo build --target wasm32-wasi --release

FROM scratch AS server
ENTRYPOINT [ "wasmedge_hyper_server.wasm" ]
COPY --from=buildserver /src/target/wasm32-wasi/release/wasmedge_hyper_server.wasm wasmedge_hyper_server.wasm
```

### serving

When the value of the `spec.workloadRuntime` field is `wasmedge` or the annotations of the Function CR contains `module.wasm.image/variant: compat-smart`:
- If `spec.serving.annotations` does not contain `module.wasm.image/variant`, `module.wasm.image/variant: compat-smart` will be automatically generated into `spec.serving.annotations`
- If `spec.serving.template.runtimeClassName` field is not set, the value of this field will be automatically set to `openfunction-crun`

#### WASM cases in Knative Serving

```yaml=
apiVersion: core.openfunction.io/v1beta2
kind: Function
metadata:
  name: wasmedge-http-server-serving
  # Optional
  # annotations:
  #   module.wasm.image/variant: compat-smart
spec:
  workloadRuntime: wasmedge
  image: openfunctiondev/wasmedge_http_server:0.1.0
  imageCredentials:
    name: push-secret
  port: 8080
  route:
    rules:
    - matches:
      - path:
          type: PathPrefix
          value: /echo
  serving:
    # Optional, user can set this field to override the default annotations
    # annotations:
    #   module.wasm.image/variant: compat-smart
    runtime: knative
    scaleOptions:
      minReplicas: 0
      maxReplicas: 10
    template:
      # Optional, user can set this field to override the default runtimeClassName
      # runtimeClassName: openfunction-crun
      containers:
      - command:
        - /wasmedge_hyper_server.wasm
        imagePullPolicy: IfNotPresent
        livenessProbe:
          initialDelaySeconds: 3
          periodSeconds: 30
          tcpSocket:
            port: 8080
        name: function
```

### BaaS

OpenFunction simply integration with various backend services (BaaS) through Dapr. Please refer to [dapr-sdk-wasi](https://github.com/second-state/dapr-sdk-wasi) and [dapr-wasm](https://github.com/second-state/dapr-wasm).

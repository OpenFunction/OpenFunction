# Dapr Proxy

## Motivation

Currently, OpenFunction injects the dapr sidecar into each Function pod, which leads to following problems：

- Function pod startup time is affected by the dapr sidecar, in my environment the startup time of the dapr sidecar is
  around 7 seconds.
- The `sidecar` mode is an blocking issue when we want to use the Function Pool to optimize the cold start speed of the
  Function.
- The `sidecar` mode means more resource overhead, which is more asymmetric for simple Functions.

So, OpenFunction should support `dapr-proxy` mode.

## Goals

- Function Pod startup time is no longer affected by the dapr sidecar.
- Resolve blocking issue for OpenFunction's cold start optimization.
- Reduce the resource overhead of Function.

## Proposal

### Function CR

Once a Function CR with `openfunction.io/dapr-service-mode: "standalone"` and `openfunction.io/enable-dapr: "true"` in the `spec.serving.annotations` field is
created, the function controller will:

- Create Dapr Proxy Deployment with annotation `dapr.io/enabled: "true"`. Set these environment variables: `FUNC_CONTEXT`, `APP_PROTOCOL`.
- Create Function Deployment with annotation `dapr.io/enabled: "false"`. Set these environment variables: `FUNC_CONTEXT`, `APP_PROTOCOL`, `DAPR_HOST`.
- Create Function Service to accept event(for async functions).

```yaml
apiVersion: core.openfunction.io/v1beta2
kind: Function
metadata:
  name: sink
spec:
  version: "v1.0.0"
  image: "openfunction/sink-sample:latest"
  port: 8080
  serving:
    annotations:
      openfunction.io/dapr-service-mode: "standalone"
      openfunction.io/enable-dapr: "true"
    runtime: "knative"
    template:
      containers:
        - name: function
          imagePullPolicy: Always
```

### Functions Framework

Adjustments that need to be made:

1. To support dapr-proxy mode, functions-framework need to get `DAPR_HOST` from env vars, init daprClient
   based on `DAPR_HOST` and `DAPR_HTTP_PORT`/`DAPR_GRPC_PORT`.
2. Do not generate/init daprClient when the sync function uses the OpenFunction signature but does not define inputs and
   outputs.

### Proxy

The implementation of proxy requires:

- The [blockUntilAppIsReady](https://github.com/dapr/dapr/blob/f5a5acc406302f0d5122ae30d18f9baba6dba8d3/pkg/runtime/runtime.go#L507)
method is called when the dapr runtime starts, so we need proxy to listen on app-port.
- The proxy is responsible for forwarding events to the function's service(only for async functions).
- The proxy should support both `GRPC` and `HTTP` protocol.
- The proxy should support load balancing.

![](https://i.imgur.com/4r6jm8x.png)

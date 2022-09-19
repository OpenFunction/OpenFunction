# One Dapr sidecar per Function

## Motivation
Currently, OpenFunction injects the dapr sidecar into each Function pod，which leads to following problems：
- Function pod startup time is affected by the dapr sidecar, in my environment the startup time of the dapr sidecar is around 7 seconds.
- The `one-sidecar-per-pod` mode is an blocking issue when we want to use the Function Pool to optimize the cold start speed of the Function (dapr runtime does not support dynamically loading components).
- The `one-sidecar-per-pod` mode means more resource overhead, which is more asymmetric for simple Functions.

So, OpenFunction should support `one dapr sidecar per function` mode.

## Goals

- Function Pod startup time is no longer affected by the dapr sidecar.
- Resolve blocking issue for OpenFunction's cold start optimization.
- Reduce the resource overhead of Function.

## Proposal
### Function CR

Once a Function CR with `dapr.io/inject-mode: "function"` in the `spec.serving.annotations` field is created, the function controller will:
- Create Function Agent Deployment with `dapr injector` related annotations. Pass function address by Function Context.
- Create Function Agent Service to accept dapr client request.
- Create Function Deployment without `dapr injector` related annotations. Pass daprd address by Function Context.
- Create Function Service to accept event(for async functions).
- Generate `APP_API_TOKEN`, configure `dapr.io/app-token-secret: "app-api-token"` and Function Deployment's env

```yaml
apiVersion: core.openfunction.io/v1beta1
kind: Function
metadata:
  name: sink
spec:
  version: "v1.0.0"
  image: "openfunction/sink-sample:latest"
  port: 8080
  serving:
    annotations:
      dapr.io/inject-mode: "function"
    runtime: "knative"
    template:
      containers:
        - name: function
          imagePullPolicy: Always
```

### Functions Framework

Adjustments that need to be made:

1. To support one-sidecar-per-function mode, functions-framework need to get daprHost from env vars, init daprClient based on daprHost and daprHTTPPort/daprGRPCPort.
2. Do not generate/init daprClient when the sync function uses the OpenFunction signature but does not define inputs and outputs.

### Function Agent
- The [blockUntilAppIsReady](https://github.com/dapr/dapr/blob/f5a5acc406302f0d5122ae30d18f9baba6dba8d3/pkg/runtime/runtime.go#L507) method is called when the dapr runtime starts, so we need a agent to listen on app-port.
- The agent is also responsible for forwarding subsequent events to the function's service(only for async funtions).
- The agent will support reconnection after disconnection.
- The agent will handle `APP_API_TOKEN` in requests.

![](https://i.imgur.com/InNUCQh.png)

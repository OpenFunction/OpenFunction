## Motivation
Tracing is important to functions. We need to support popular tracing technologies like SkyWalking and OpenTelemetry.

## Goals
Users can select to use SkyWalking/OpenTelemetry for tracing or turn function tracing off. Tracing code and options should be added in the function framework instead of user functions.

## Proposal
### Global level tracing
Global level tracing plugin setting can be added to a configmap named `openfunction-config` in the `openfunction` namespace:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  labels:
    app.kubernetes.io/version: v0.5.0
  name: openfunction-config
  namespace: openfunction
data:
  plugins.tracing: |
    # Switch for tracing, default to false
    enabled: true
    # Provider name can be set to "skywalking", "opentelemetry"
    # A valid provider must be set if tracing is enabled.
    provider: 
      name: "skywalking"
      oapServer: "localhost:xxx"
    # Custom tags to add to tracing
    tags:
    - func: function-with-tracing
    - layer: faas
    - tag1: value1
    - tag2: value2
    baggage:
    # baggage key is `sw8-correlation` for skywalking and `baggage` for opentelemetry
    # Correlation context for skywalking: https://skywalking.apache.org/docs/main/latest/en/protocols/skywalking-cross-process-correlation-headers-protocol-v1/
    # baggage for opentelemetry: https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/baggage/api.md
    # W3C Baggage Specification/: https://w3c.github.io/baggage/
      key: sw8-correlation # key should be baggage for opentelemetry
      value: "base64(string key):base64(string value),base64(string key2):base64(string value2)"
```
### Function level tracing
Function level tracing plugin setting can be added to the annotation of the Serving CRD and pass these options via function context to function framework:

```yaml
apiVersion: core.openfunction.io/v1alpha2
kind: Function
metadata:
  name: function-with-tracing
  annotations:
    plugins.tracing: |
      # Switch for tracing, default to false
      enabled: true
      # Provider name can be set to "skywalking", "opentelemetry"
      # A valid provider must be set if tracing is enabled.
      provider: 
        name: "skywalking"
        oapServer: "localhost:xxx"
      # Custom tags to add to tracing
      tags:
      - func: function-with-tracing
      - layer: faas
      - tag1: value1
      - tag2: value2
      baggage:
      # baggage key is `sw8-correlation` for skywalking and `baggage` for opentelemetry
      # Correlation context for skywalking: https://skywalking.apache.org/docs/main/latest/en/protocols/skywalking-cross-process-correlation-headers-protocol-v1/
      # baggage for opentelemetry: https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/baggage/api.md
      # W3C Baggage Specification/: https://w3c.github.io/baggage/
        key: sw8-correlation # key should be baggage for opentelemetry
        value: "base64(string key):base64(string value),base64(string key2):base64(string value2)"
     
spec:
  serving:
    runtime: "OpenFuncAsync"
```

- Changes to function context:
```json
{
  "name": "function-with-tracing",
  "version": "v1",
  "requestID": "a0f2ad8d-5062-4812-91e9-95416489fb01",
  "port": "50002",
  "inputs": {},
  "outputs": {},
  "runtime": "OpenFuncAsync",
  "state": "",
  "plugins.tracing": {
    "enabled": true,
    "provider": {
      "name": "skywalking",
      "oapServer": "localhost:xxx"
    },
    "tags": [
      {
        "func": "function-with-tracing"
      },
      {
        "layer": "faas"
      },
      {
        "tag1": "value1"
      },
      {
        "tag2": "value2"
      }
    ],
    "baggage": {
      "key": "sw8-correlation",
      "value": "base64(string key):base64(string value),base64(string key2):base64(string value2)"
    }
  }
}
```
# OpenFunction Gateway

## Motivation
The new Gateway APIs aim to take the learnings from various Kubernetes ingress implementations, to build a standardized vendor neutral API.

It provides some advanced features like HTTP traffic splitting, Cross-Namespace routing, etc. which is powerful and valuable for FaaS as well to be the function's API Gateway.

## Goals
- OpenFucntion developers can use any downstream implementations of the Gateway API
- OpenFucntion users needn't deploy extra ingress controller
- OpenFunction will no longer create a CRD domain that only itself can understand
- Leverage the power of the Gateway API to provide better networking capabilities for openfunction

## Proposal
### Domain
Since the domain will be deprecated, the following resources will be removed:
- The Domain CRD will be removed
- The Domain-related logic will be removed
- The `OpenFunction/config/domain/default-domain.yaml` will be removed
```yaml=
apiVersion: core.openfunction.io/v1beta1
kind: Domain
metadata:
  name: openfunction
  namespace: io
spec:
  ingress:
    annotations:
      nginx.ingress.kubernetes.io/rewrite-target: /$2
      nginx.ingress.kubernetes.io/upstream-vhost: $service_name.$namespace
    ingressClassName: nginx
    service:
      name: ingress-nginx-controller
      namespace: ingress-nginx
```
### Unified access entry
To support a unified access entry, the gateway controller will:
- When an `gateway.networking.openfunction.io` CR have been created, lookup this service: `gateway.openfunction.svc.cluster.local`
- If the above service is not found, gateway controller will create this service based on the `gatewayRef` or `gatewayDef` field
- The service will create a cname to the target gateway, and we can access the function through a unified entry, regardless of the real gateway address and which provider it is implemented by
```yaml=
apiVersion: v1
kind: Service
metadata:
  name: gateway
  namespace: openfunction
spec:
  # contour
  # externalName: envoy.projectcontour.svc.cluster.local
  # istio
  externalName: gateway.istio-ingress.svc.cluster.local
  # ports will sync from target service
  ports:
  - name: http
    port: 80
    protocol: TCP
    targetPort: 80
  - name: https
    port: 443
    protocol: TCP
    targetPort: 443
  sessionAffinity: None
  type: ExternalName
status:
  loadBalancer: {}
```
#### Usage
##### Use function service address to access function

Assume you have a function named `function-sample` in namespace `default` with a path of `/get`, we can access function via:
```shell=
curl -sv -I "http://function-sample.default.svc.cluster.local/get"
```
This address will be used as function internal address, it's suitable as `sink.url` of `EventSource`.
##### Use openfunction gateway address to access function
- for Path-Based routing we can access function via:
```shell=
curl -sv -I -HHost:<prefix>.ofn.io "http://gateway.openfunction.svc.cluster.local/path/to/service"
```
- for Host-Based routing we access function via:
```shell=
curl -sv -I -HHost:[someprefixs].ofn.io "http://gateway.openfunction.svc.cluster.local"
```
- we can also use both Host-Based and Path-Based to access function via:
```shell=
curl -sv -I -HHost:[someprefixs].ofn.io "http://gateway.openfunction.svc.cluster.local/path/to/service"
```
##### Use the domain defined in the gateway to access function
1. Configure `CoreDns` based on the domain in the `gateway.networking.openfunction.io` CR and openfunction gateway address. for example, for a gateway with the domain `*.ofn.io`, will modify the configuration of `CoreDns` like this:
```shell=
kubectl -n kube-system edit cm coredns -o yaml
```
This line will be added to the configuration file in the `.:53` section:
`rewrite stop name regex .*\.ofn\.io gateway.openfunction.svc.cluster.local` 
```yaml=
apiVersion: v1
data:
  Corefile: |
    .:53 {
        errors
        health {
           lameduck 5s
        }
        ready
        kubernetes cluster.local in-addr.arpa ip6.arpa {
           pods insecure
           fallthrough in-addr.arpa ip6.arpa
           ttl 30
        }
        rewrite stop name regex .*\.ofn\.io gateway.openfunction.svc.cluster.local
        prometheus :9153
        forward . /etc/resolv.conf {
           max_concurrent 1000
        }
        cache 30
        loop
        reload
        loadbalance
    }
...
```
2. If the user is also using `nodelocaldns`, such as the `kubesphere` user, will modify the configuration of `nodelocaldns` like this:
```shell=
kubectl -n kube-system edit cm nodelocaldns -o yaml
```
```yaml=
apiVersion: v1
data:
  Corefile: |
    ofn.io:53 {
        errors
        cache {
            success 9984 30
            denial 9984 5
        }
        reload
        loop
        bind 169.254.25.10
        forward . 10.233.0.3 {
            force_tcp
        }
        prometheus :9253
    }
    cluster.local:53 {
        errors
        cache {
            success 9984 30
            denial 9984 5
        }
        reload
        loop
        bind 169.254.25.10
        forward . 10.233.0.3 {
            force_tcp
        }
        prometheus :9253
        health 169.254.25.10:9254
    }
    in-addr.arpa:53 {
        errors
        cache 30
        reload
        loop
        bind 169.254.25.10
        forward . 10.233.0.3 {
            force_tcp
        }
        prometheus :9253
    }
    ip6.arpa:53 {
        errors
        cache 30
        reload
        loop
        bind 169.254.25.10
        forward . 10.233.0.3 {
            force_tcp
        }
        prometheus :9253
    }
    .:53 {
        errors
        cache 30
        reload
        loop
        bind 169.254.25.10
        forward . /etc/resolv.conf
        prometheus :9253
    }
...
```
3. Now, we can Use the domain defined in the gateway to access function:
- For Path-Based routing we can access function via:
```shell=
curl -sv -I http://<prefix>.ofn.io/path/to/service
```
- For Host-Based routing we can access function via:
```shell=
curl -sv -I http://[someprefixs].ofn.io
```
- We can also use both Host-Based and Path-Based to access function like this:
```shell=
curl -sv -I http://[someprefixs].ofn.io/path/to/service
```

### Gateway
Whenever a gateway is created, the gateway controller will:
- Generate the corresponding `gateway.networking.k8s.io` CR based on the `gatewayDef` and `gatewaySpec`
- Or find the corresponding `gateway.networking.k8s.io` CR based on the `gatewayRef`, then reconcile the corresponding `gateway.networking.k8s.io` CR
- Lookup or create service `gateway.openfunction.svc.cluster.local`,  the service will create a cname to the target gateway based on the `gatewayRef` or `gatewayDef` field

Whenever a gateway have changed, the gateway controller will:
- When `gatewayDef` field have changed, update the corresponding `gateway.networking.k8s.io` CR based on the `gatewayDef` and `gatewaySpec`
- When `gatewayRef` field have changed, reconcile the corresponding `gateway.networking.k8s.io` based on the `gatewayRef` and `gatewaySpec`
- When `hostTemplate`, `pathTemplate` fields have changed, the function controller will update the `HTTPRoute`
- When other `gateway.networking.k8s.io` CR related fields have changed, reconcile the corresponding `gateway.networking.k8s.io` CR
```yaml=
apiVersion: networking.openfunction.io/v1beta1
Kind: Gateway
metadata:
  name: openfunction
  namespace: openfunction
spec:
  # Used to generate the hostname field of gatewaySpec.listeners.openfunction.hostname
  domain: ofn.io
  # Used to generate the hostname field of gatewaySpec.listeners.openfunction.hostname
  # default value is cluster.local
  clusterDomain: cluster.local
  # Used to generate the hostname of attaching HTTPRoute
  hostTemplate: "{{.Name}}.{{.Namespace}}.{{.Domain}}"
  # Used to generate the path of attaching HTTPRoute
  pathTemplate: "{{.Namespace}}/{{.Name}}"
  # Label key to add to the HTTPRoute generated by function, the value will be the `gateway.openfunction.openfunction.io` CR's namespaced name
  httpRouteLabelKey: "app.kubernetes.io/managed-by"
  # Reference to an existing K8s gateway  
  gatewayRef:
    name: openfunction
    namespace: contour
  # Definition to a new K8s gateway
  gatewayDef:
    # The name is the same as metadata.name
    namespace: istio-ingress
    gatewayClassName: istio
  # Gateway listener item to add for OpenFunction
  # Can be added to an existing K8s gateway or a new created one
  gatewaySpec:
    listeners:
    - name: ofn-http-internal
      # The hostname is used for function internal address
      # generated from *.<clusterDomain>
      # hostname: "*.cluster.local"
      protocol: HTTP
      port: 80
      allowedRoutes:
        namespaces:
          from: All
    - name: ofn-http-external
      # The hostname is generated from *.<domain>
      # hostname: "*.ofn.io"
      protocol: HTTP
      port: 80
      allowedRoutes:
        namespaces:
          from: All
    - name: ofn-https-external
      # The hostname is generated from *.<domain>
      # hostname: "*.ofn.io"
      protocol: HTTPS
      port: 443
      allowedRoutes:
        namespaces:
          from: All
      tls:
        certificateRefs:
        - name: openfunction-cert
status:
  # watch the `gateway.networking.k8s.io` CR and sync its status
  addresses:
  - type: Hostname
    value: gateway.istio-ingress.svc.cluster.local:80
  - type: Hostname
    value: gateway.istio-ingress.svc.cluster.local:443
  conditions:
  - lastTransitionTime: "2022-06-12T14:15:48Z"
    message: Gateway valid, assigned to service(s) gateway.istio-ingress.svc.cluster.local:80
    observedGeneration: 2
    reason: ListenersValid
    status: "True"
    type: Ready
  listeners:
  - attachedRoutes: 2
    conditions:
    - lastTransitionTime: "2022-06-16T14:48:32Z"
      message: No errors found
      observedGeneration: 1
      reason: ListenerReady
      status: "True"
      type: ResolvedRefs
    name: ofn-http
    supportedKinds:
    - group: gateway.networking.k8s.io
      kind: HTTPRoute
```
### Route
Whenever a function is created, the function controller will:
- Look for the `gateway.networking.openfunction.io` CR called `openfunction` in `openfunction` namespace if `route.gatewayRef` is not defined
- Look for the `gateway.networking.openfunction.io` CR in the specific namespace with specific name if route.gatewayRef` is defined
- Watch the `gateway.networking.openfunction.io` CR
- Based on the content of the `gateway.networking.openfunction.io` CR, generate or modify the `httproute.gateway.networking.k8s.io` CR
- The `hostnames` of the `httproute.gateway.networking.k8s.io` CR is generated from `hostTemplate` only if the user does not specify a hostname
- The `path` of the `httproute.gateway.networking.k8s.io` CR is generated from `pathTemplate` only if the user does not specify a path
- The `BackendRefs` of the `httproute.gateway.networking.k8s.io` CR will be automatically link to the corresponding k8s service
- And label `HTTPRouteLabelKey` should be added to the `httproute.gateway.networking.k8s.io` CR whose value is the `gateway.openfunction.openfunction.io` CR's namespaced name
- Create a new HTTPRoute CR, then watch and sync its status
- Create service `{{.Name}}.{{.Namespace}}.svc.cluster.local`, the service will create a cname to OpenFunction gateway. This address will be used as function internal address

```yaml=
apiVersion: core.openfunction.io/v1beta1
kind: Function
metadata:
  name: function-sample
  labels:
    app: openfunction
spec:
  version: "v2.0.0"
  image: "openfunctiondev/sample-go-func:latest"
  imageCredentials:
    name: push-secret
  port: 8080 # default to 8080
  route:
    ## The function controller will try to find the specified gateway CR in the specified namespace 
    ## If gatewayRef is not specified, will try to find the openfunction/openfunction gateway CR
    gatewayRef:
      name: openfunction
      namespace: openfunction
#    The `hostnames` is generated based on hostTemplate field of the
#    gateway.networking.openfunction.io CR specified by gatewayRef
#    User can specify hostnames manually to override the global default values specified in gateway.networking.openfunction.io CR 
#    hostnames: 
#    - "function-sample.default.ofn.io"
#    will be auto generated base on `{{.Name}}.{{.Namespace}}.svc.cluster.local`
#    This hostname is not bound by host-template, host-template only affects domain-related hostnames
#    - "function-sample.default.svc.cluster.local"
#    rules:
#    - matches:
#      - path:
#        type: PathPrefix
         # The `path` is generated based on pathTemplate field of the gateway.networking.openfunction.io CR specified by gatewayRef
         # User can specify pathTemplate manually to override the global default values specified in gateway.networking.openfunction.io CR 
#        value: /default/function-sample
#      filters:
#        - type: RequestHeaderModifier
#        requestHeaderModifier:
#          add:
#          - name: my-added-header
#            value: added-value
#    traffic:
#      knative:
#      - latestRevision: true
#        percent: 50
#      - latestRevision: false
#        percent: 50
#        revisionName: function-sample-00001
  build:
    builder: openfunction/builder-go:latest
    env:
      FUNC_NAME: "HelloWorld"
      FUNC_CLEAR_SOURCE: "true"
    srcRepo:
      url: "https://github.com/OpenFunction/samples.git"
      sourceSubPath: "functions/knative/hello-world-go"
      revision: "main"
  serving:
    template:
      containers:
        - name: function
          imagePullPolicy: Always
    runtime: "knative"
status:
  # holds the addresses that used to access the Function.
  addresses:
  - type: internal
    # generated by gateway address and paths
    value: http://function-sample.default.svc.cluster.local:80/default/function-sample
  - type: external
    # if dns configured, we can use this address to access function
    # generated by domain, listener's port, hosts and paths.
    value: http://function-sample.default.ofn.io:80/default/function-sample
  build: 
  serving:
  route:
    hosts: 
    - function-sample.default.ofn.io
    - function-sample.default.svc.cluster.local
    paths: 
    - type: PathPrefix
      value: /default/function-sample
    # watch the HTTPRoute CR and sync its status
    conditions:
    - lastTransitionTime: "2022-06-16T06:45:58Z"
      message: Route was valid
      observedGeneration: 2
      reason: RouteAdmitted
      status: "True"
      type: Accepted
```

### Native Gateway Resource Provisioning mode
- Users provide their own `gateway.networking.k8s.io` CR
The user create or use an already existing the `gateway.networking.k8s.io` CR, and then specifies in the `gatewayRef` field of the `gateway.networking.openfunction.io` CR.

- The user create a `gateway.networking.k8s.io` CR through openfunction
The user create the `gateway.networking.openfunction.io` CR with `gatewayDef` and `gatewaySpec`, then OpenFunction will create corresponding `gateway.networking.k8s.io` CR

- OpenFunction provides default `gateway.networking.k8s.io` CR
OpenFunction will install (and can be disabled) a gateway api implementation by default, perhaps Contour, and then creates a gateway CR as the default gateway, which the user does not have to specify in the `gatewayRef` field of the function CR.


### Traffic Mechanisms
- Orignal
  ![](https://i.imgur.com/lAzF2nH.png)
- Gateway
  ![](https://i.imgur.com/6Ri5F6G.png)


## Action Items
- Update the function CRD
- Add the gateway CRD
- Remove Domain-related resources
- Implementing the route part logic of the function controller
- Implementing the gateway controller
- Update API conversion
- Update e2e tests
- Add example & Update documentation

## Reference
- [K8s Gateway API](https://gateway-api.sigs.k8s.io/v1alpha2/guides/getting-started/)
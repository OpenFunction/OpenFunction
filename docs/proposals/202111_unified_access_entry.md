## Motivation

Use ingress to create a unified access entry for Function.

## Design

### Plan A

Step 1: Create an external service for each Function that points to the ingress controller's service.

```yaml
apiVersion: v1
kind: Service
metadata:
name: function-sample
namespace: default
spec:
externalName: ingress-nginx-controller.ingress-nginx.svc.cluster.local
internalTrafficPolicy: Cluster
ports:
- name: http2
  port: 80
  protocol: TCP
  targetPort: 80
  sessionAffinity: None
  type: ExternalName
  status:
  loadBalancer: {}
```

Step 2: All functions in the same namespace will use the same ingress.
The function controller will add a corresponding `path` to the ingress whenever the function is created,
and update the `path` whenever the function is updated.

The ingress looks like this.
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
annotations:
nginx.ingress.kubernetes.io/upstream-vhost: $service_name.$namespace
name: function-sample
namespace: default
spec:
  ingressClassName: nginx
  rules:
  - host: function-sample.default
    http:
      paths:
      - backend:
        service:
          name: function-sample-serving-d55zz-ksvc-gd64w
          port:
            number: 80
        path: /
        pathType: Prefix
status:
 loadBalancer: {}
```

Step 3: Use

```
http://<Function name>.<Function namespace>/
```

to access the Function.

### Plan B

Step 1: Create a service named `openfunction` in the same namespace as the ingress controller, for example `ingress-nginx`.  The endpoints of the service are the ingress controller pod.

Step 2: All functions in the same namespace will use the same ingress.
The function controller will add a corresponding `path` to the ingress whenever the function is created,
and update the `path` whenever the function is updated.

The ingress looks like this.

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    nginx.ingress.kubernetes.io/upstream-vhost: $service_name.$namespace
    nginx.ingress.kubernetes.io/rewrite-target: /$2
  name: function-sample
  namespace: default
spec:
  ingressClassName: nginx
  rules:
  - http:
      paths:
      - backend:
          service:
            name: serving-d55zz-ksvc-gd64w
            port:
              number: 80
        path: /default/function-sample(/|$)(.*)
        pathType: Prefix
status:
  loadBalancer: {}
```

Step 3: Use

```shell
http://<Service name>.<Ingress controller namespace>/<Function namespace>/<Function name>/
``` 
to access the Function.

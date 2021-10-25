## Motivation

Use ingress to create a unified access entry for Function.

## Design

### Plan A

Firstly, create an external service which direction to the Ingress Controller for each Function.

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

Secondly, all Functions in the same namespace will use the same ingress.
The Function operator will add a corresponding `path` to the ingress when the Function created,
and update the `path` when the Function updated.

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

Thirdly, use

```
http://<Function name>.<Function namespace>/
```

to access the Function.

  

### Plan B

Firstly, create a service such as `openfunction` in the namespace which ingress controller be in such as `ingress-nginx`, 
the endpoints of the service are the ingress controller pod.

Secondly, all Functions in the same namespace will use the same ingress.
The Function operator will add a corresponding `path` to the ingress when the Function created,
and update the `path` when the Function updated.

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

Thirdly, use 

```shell
http://<Service name>.<Ingress controller namespace>/<Function namespace>/<Function name>/
``` 
to access the Function.

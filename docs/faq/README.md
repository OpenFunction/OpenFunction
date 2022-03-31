# FAQ

## Q: How to use private image repositories in OpenFunction?

A: OpenFunction uses Shipwright (which utilizes Tekton to integrate with Cloud Native Buildpacks) in the build phase to package the user function to the application image.

Users often choose to access a private image repository in an insecure way, which is not yet supported by the Cloud Native Buildpacks.

We offer a workaround as below to get around this limitation for now:

1. Use IP address instead of hostname as access address for private image repository.
2. You should [skip tag-resolution](https://knative.dev/docs/serving/configuration/deployment/#skipping-tag-resolution) when you run the Knative-runtime function.

For references:

[buildpacks/lifecycle#524](https://github.com/buildpacks/lifecycle/issues/524)

[buildpacks/tekton-integration#31](https://github.com/buildpacks/tekton-integration/issues/31)

## Q: How to access the Knative-runtime function without introducing a new ingress controller?

A: OpenFunction provides a unified entry point for function accessibility, which is based on the Ingress Nginx implementation. However, for some users, this is not necessary, and instead, introducing a new ingress controller may affect the current cluster.

In general, accessible addresses are for the sync(Knative-runtime) functions. Here are two ways to solve this problem:

- Magic DNS

  You can follow [this guide](https://knative.dev/docs/install/yaml-install/serving/install-serving-with-yaml/#configure-dns) to config the DNS.

- CoreDNS

  This is similar to using Magic DNS, with the difference that the configuration for DNS resolution is placed inside CoreDNS. Assume that the user has configured a domain named "openfunction.dev" in the ConfigMap `config-domain` under the `knative-serving` namespace (as shown below):

  ```shell
  $ kubectl -n knative-serving get cm config-domain -o yaml
  
  apiVersion: v1
  data:
    openfunction.dev: ""
  kind: ConfigMap
  metadata:
    annotations:
      knative.dev/example-checksum: 81552d0b
    labels:
      app.kubernetes.io/part-of: knative-serving
      app.kubernetes.io/version: 1.0.1
      serving.knative.dev/release: v1.0.1
    name: config-domain
    namespace: knative-serving
  ```

  Next, let's add an A record for this domain. OpenFunction uses Kourier as the default network layer for Knative Serving, which is where the domain should flow to.

  ```shell
  $ kubectl -n kourier-system get svc
  
  NAME               TYPE           CLUSTER-IP     EXTERNAL-IP   PORT(S)                      AGE
  kourier            LoadBalancer   10.233.7.202   <pending>     80:31655/TCP,443:30980/TCP   36m
  kourier-internal   ClusterIP      10.233.47.71   <none>        80/TCP                       36m
  ```
  
  Then the user only needs to configure this Wild-Card DNS resolution in CoreDNS to resolve the URL address of any Knative Service in the cluster.
  
  > Where "10.233.47.71" is the address of the Service kourier-internal.
  
  ```shell
  $ kubectl -n kube-system get cm coredns -o yaml
  
  apiVersion: v1
  data:
    Corefile: |
      .:53 {
          errors
          health
          ready
          template IN A openfunction.dev {
            match .*\.openfunction\.dev
            answer "{{ .Name }} 60 IN A 10.233.47.71"
            fallthrough
          }
          kubernetes cluster.local in-addr.arpa ip6.arpa {
            pods insecure
            fallthrough in-addr.arpa ip6.arpa
          }
          hosts /etc/coredns/NodeHosts {
            ttl 60
            reload 15s
            fallthrough
          }
          prometheus :9153
          forward . /etc/resolv.conf
          cache 30
          loop
          reload
          loadbalance
      }
      ...
  ```
  
  If the user cannot resolve the URL address for this function outside the cluster, configure the `hosts` file as follows:
  
  > Where "serving-sr5v2-ksvc-sbtgr.default.openfunction.dev" is the URL address obtained from the command "kubectl get ksvc".
  
  ```shell
  10.233.47.71 serving-sr5v2-ksvc-sbtgr.default.openfunction.dev
  ```

After the above configuration is done, you can get the URL address of the function with the following command. Then you can trigger the function via `curl` or your browser.

```shell
$ kubectl get ksvc

NAME                       URL
serving-sr5v2-ksvc-sbtgr   http://serving-sr5v2-ksvc-sbtgr.default.openfunction.dev
```


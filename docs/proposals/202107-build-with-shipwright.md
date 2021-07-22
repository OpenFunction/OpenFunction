# Build with Shipwright in OpenFunction


## Shipwright
Shipwright is an extensible framework for building container images on Kubernetes.

Shipwright supports any tool that can build container images in Kubernetes clusters, such as:

- Kaniko
- Cloud Native Buildpacks
- BuildKit
- Buildah

## Integrated

Shipwright can build images through Dockerfile, which can greatly enrich the ability of OpenFunction.

Now the Shipwright is at a relatively early stage, many functions are not perfect. Based on this fact, I designed two different methods to integrated Shipwright to OpenFunction.

### Plan A: Use Shipwright to manage build

Now the OpenFunction used tekton to build images. OpenFunction controller needs to manage the task, taskrun, pipeline, pipelinerun and other objects. The Shipwright encapsulates tekton, simplifying these objects into `Build`, `BuildRun`, and `Strategy`. 

We can use Shipwright to build the image to reduce the work of the controller. 

Firstly, define a `ClusterBuildStrategy` named `openfunction` like this.

```yaml=
apiVersion: shipwright.io/v1alpha1
kind: ClusterBuildStrategy
metadata:
  name: openfunction
spec:
  parameters:
    - name: APP_IMAGE
      description: The name of where to store the app image.
    - name: SOURCE_SUBPATH
      description: A subpath within the source input where the source to build is located.
      default: ""
    - name: PROCESS_TYPE
      description: The default process type to set on the image.
      default: "web"
    - name: RUN_IMAGE
      description: Reference to a run image to use.
      default: ""
    - name: CACHE_IMAGE
      description: The name of the persistent app cache image (if no cache workspace is provided).
      default: ""
    - name: SKIP_RESTORE
      description: Do not write layer metadata or restore cached layers.
      default: "false"
    - name: USER_ID
      description: The user ID of the builder image user.
      default: "1000"
    - name: GROUP_ID
      description: The group ID of the builder image user.
      default: "1000"
    - name: PLATFORM_DIR
      description: The name of the platform directory.
      default: empty-dir
  buildSteps:
    - name: prepare
      image: docker.io/library/bash:5.1.4@sha256:b208215a4655538be652b2769d82e576bc4d0a2bb132144c060efc5be8c3f5d6
      command: ["/bin/sh"]
      args:
        - -c
        - |
          #!/usr/bin/env bash
          set -e

          for path in "/cache" "/tekton/home" "/layers" "/workspace/source"; do
            echo "> Setting permissions on '$path'..."
            chown -R "$(params.USER_ID):$(params.GROUP_ID)" "$path"
          done
      volumeMounts:
        - name: cache
          mountPath: /cache
        - name: layers-dir
          mountPath: /layers
        - name: $(params.PLATFORM_DIR)
          mountPath: /platform
      securityContext:
        privileged: true

    - name: create
      image: openfunction/builder:v1
      imagePullPolicy: Always
      command: [ "/cnb/lifecycle/creator" ]
      args:
        - "-app=/workspace/source/$(params.SOURCE_SUBPATH)"
        - "-cache-dir=/cache"
        - "-uid=$(params.USER_ID)"
        - "-gid=$(params.GROUP_ID)"
        - "-layers=/layers"
        - "-platform=/platform"
        - "-report=/layers/report.toml"
        - "-process-type=$(params.PROCESS_TYPE)"
        - "-skip-restore=$(params.SKIP_RESTORE)"
        - "-previous-image=$(params.APP_IMAGE)"
        - "-run-image=$(params.RUN_IMAGE)"
        - "$(params.APP_IMAGE)"
      volumeMounts:
        - name: cache
          mountPath: /cache
        - name: layers-dir
          mountPath: /layers
        - name: $(params.PLATFORM_DIR)
          mountPath: /platform
      securityContext:
        runAsUser: 1000
        runAsGroup: 1000

    - name: results
      image: docker.io/library/bash:5.1.4@sha256:b208215a4655538be652b2769d82e576bc4d0a2bb132144c060efc5be8c3f5d6
      command: ["/bin/sh"]
      args:
        - -c
        - |
          #!/usr/bin/env bash
          set -e
          cat /layers/report.toml | grep "digest" | cut -d'"' -f2 | cut -d'"' -f2 | tr -d '\n' | tee $(results.shp-image-digest.path)
      volumeMounts:
        - name: layers-dir
          mountPath: /layers
```

Secondly, create a `Build`.

```yaml=
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: buildpack-hello-world
spec:
  source:
    url: https://github.com/OpenFunction/samples
    contextDir: functions-framework/Knative/simple-hello/userfunction
  strategy:
    name: openfunction
    kind: ClusterBuildStrategy
  output:
    image: docker.io/openfunctiondev/hellow-world:latest
    credentials:
      name: push-secret
  paramValues:
    - name: APP_IMAGE
      value: docker.io/openfunctiondev/hellow-world:latest
```

The `push-secret` is a secret to access your container registry, you can create it like this.

```shell
REGISTRY_SERVER=https://index.docker.io/v1/ REGISTRY_USER=<your_registry_user> REGISTRY_PASSWORD=<your_registry_password>
kubectl create secret docker-registry push-secret \
    --docker-server=$REGISTRY_SERVER \
    --docker-username=$REGISTRY_USER \
    --docker-password=$REGISTRY_PASSWORD  \
    --docker-email=<your_email>
```

Thirdly, create a `BuildRun` to start build the image.

```yaml=
apiVersion: shipwright.io/v1alpha1
kind: BuildRun
metadata:
  generateName: buildpack-hellow-world-buildrun-
spec:
  buildRef:
    name: buildpack-hellow-world
```

The `Function` CRD needs to change to this.

```yaml=
apiVersion: core.openfunction.io/v1alpha1
kind: Function
metadata:
  name: function-sample
spec:
  version: "v1.0.0"
  image: "openfunctiondev/sample-go-func:latest"
  #port: 8080 # default to 8080
  build:
    strategy: Dockerfile
    dockerfile: cmd/Dockerfile
    params:
      FUNC_NAME: "HelloWorld"
      FUNC_TYPE: "http"
      #FUNC_SRC: "main.py"
    srcRepo:
      url: "https://github.com/OpenFunction/samples.git"
      sourceSubPath: "functions/Knative/hello-world-go"
    registry:
      url: "https://index.docker.io/v1/"
      credentials:
        name: push-secret
```

| key | Describe | Type |
| -------- | -------- | -------- |
| strategy     | The `Strategy` which used to build, `Dockerfile` or `Openfunction`     | string     |
| dockerfile     | The Dockerfile used to build image     | string     |
| credentials     | The secret to accessing your container registry     | corev1.LocalObjectReference     |


The `Builder` CRD needs to change to this.

```yaml=
apiVersion: core.openfunction.io/v1alpha1
kind: Builder
metadata:
  name: function-sample-builder
spec:
  version: "v1.0.0"
  image: "openfunctiondev/sample-go-func:latest"
  #port: 8080 # default to 8080
  strategy: Dockerfile
  dockerfile: cmd/Dockerfile
  params:
    FUNC_NAME: "HelloWorld"
    FUNC_TYPE: "http"
      #FUNC_SRC: "main.py"
  srcRepo:
    url: "https://github.com/OpenFunction/samples.git"
      sourceSubPath: "functions/Knative/hello-world-go"
  registry:
    url: "https://index.docker.io/v1/"
    credentials:
      name: push-secret
``` 

### Plan B: Use Shipwright to build Dockerfile

Keep the existing build logic unchanged, only use Shipwright to build the dockerfile.

The `Function` CRD needs to add a filed named `dokerfile`.

```yaml=
apiVersion: core.openfunction.io/v1alpha1
kind: Function
metadata:
  name: function-sample
spec:
  build:
    dockerfile: Dockerfile
```

The `dockerfile` is the Dockerfile used to build the image. When the `dockerfile` is not nil, Openfunction used Shipwright to build the image.

The `Builder` CRD needs to add the filed `dokerfile` too.

```yaml=
apiVersion: core.openfunction.io/v1alpha1
kind: Builder
metadata:
  name: function-sample-builder
spec:
  dockerfile: Dockerfile
```

- # OpenFunction

  ---

  ## Overview

  ---

  ```OpenFunction``` is a cloud-native open source FaaS (Function as a Service) platform with the main goal of enabling users to focus more on their business logic without having to relate to the business runtime environment and underlying infrastructure, and users only need to submit business-related source code in the form of functions.

  ```OpenFunction``` includes and is not limited to the following features:

  - Convert business-related function source code to runnable application source code.
  - Generate a deployable container image from the converted application source code.
  - Deploy the generated container image to the underlying runtime environment such as K8s, and automatically scale up and down according to business traffic, and scale to 0 when there is no traffic.
  - Provide event management functions for trigger functions.
  - Provide additional functions to manage function versions, traffic entry, etc.

  ![](docs/images/OpenFunction-architecture.png)

  ## Prerequisites

  ---

  The current version of OpenFunction requires that you have a Kubernetes cluster with version ``>=1.18.6``.

  In addition, you need to deploy implementation components for the OpenFunction abstraction components ``Builder``, ``Serving``.

  #### Builder

  You can choose from the following options:

  - OpenFunction Builder implementation based on Tekton and Cloud Native Buildpacks, you need to [install the Tekton project](https://tekton.dev/docs/getting-started/#installation) and then integrate it with the [Cloud Native Buildpacks project](https://buildpacks.io/docs/tools/tekton/).

  #### Serving

  You can choose from the following options:

  - OpenFunction Serving implementation based on Knative, you need to [install the Knative project](https://knative.dev/docs/install/).

  ## CustomResourceDefinitions

  ---

  The core functionality of OpenFunction is to serve users with the ability to develop, run, and manage business applications as execution units of code functions. OpenFunction is implemented based on the following [custom resource definitions (CRDs)](https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/):

  - **```Function```**, abstract definition of a function.
  - **```Builder```**, abstract definition of a function builder.
  - **```Serving```**, abstract definition of a function workload.

  ## QuickStart

  ---

  ### Install

  You can quickly build an OpenFunction platform by executing the following command after selecting an appropriate version.

  ```shell
  kubectl apply -f config/bundle.yaml
  ```

  > Note: When using non-default namespaces, make sure that the ClusterRoleBinding in the namespace is adapted.

  ### Sample: Run a function.

  If you have already built implementations of OpenFunction Builder and OpenFunction Serving, follow steps below to run a simple case.

  1. Creating a secret

     In order to access your mirror repository, you need to create a secret. You can create this secret by editing the ``username`` and ``password`` fields in ``config/samples/registry-account.yaml``, and then apply it.

     ```shell
     kubectl apply -f config/samples/registry-account.yaml
     ```

  2. Creating functions

     Here is a simple Function case for you, modify the ``spec.image`` field in ``config/samples/core_v1alpha1_function.yaml`` to your own repository address: 

     ```yaml
     apiVersion: core.openfunction.io/v1alpha1
     kind: Function
     metadata:
       name: function-sample
     spec:
       image: "<your registry name>/sample-go-func"
     ```

     Use the following command to create this Function:

     ```shell
     kubectl apply -f config/samples/core_v1alpha1_function.yaml
     ```

  3. Result observation

     You can observe the process of Function with the following command:

     ```shell
     kubectl get functions.core.openfunction.io
     
     NAME              AGE
     function-sample   5s
     ```

     You can also observe the process of Builder in the [Tekton Dashboard](https://tekton.dev/docs/dashboard/).

     Finally, you can observe the final state of the case in the Serving:

     ```shell
     kubectl get servings.core.openfunction.io
     
     NAME                      AGE
     function-sample-serving   15s
     ```

     At this point, you can view the service entry for the case function with the following command:

     ```shell
     kubectl get ksvc
     
     NAME                           URL                                                                 LATESTCREATED                        LATESTREADY                          READY   REASON
     function-sample-serving-ksvc   http://function-sample-serving-ksvc.default.<external-ip>.xip.io   function-sample-serving-ksvc-00001   function-sample-serving-ksvc-00001   True
     ```

     Or get the service address directly with the following command:

     where``` <external-ip> ```indicates the external address of your gateway service

     ```shell
     kubectl get ksvc function-sample-serving-ksvc -o jsonpath={.status.url}
     
     http://function-sample-serving-ksvc.default.<external-ip>.xip.io
     ```

     Access the above service address via commands such as ``curl``:

     ```shell
     curl http://function-sample-serving-ksvc.default.<external-ip>.xip.io
     
     Hello, World!
     ```

  ### Removal

  You can uninstall the components of OpenFunction by executing the following command:

  ```shell
  kubectl delete -f config/bundle.yaml
  ```

  ## Development

  ---

  You can get help on developing this project by visiting [Development guide](docs/development/development-guide.md).


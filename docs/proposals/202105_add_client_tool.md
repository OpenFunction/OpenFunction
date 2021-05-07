## Motivation

OpenFunction currently lacks a simple and efficient command line tool to manage
its resources.

Having an effective command line tool allows users to focus on the core functionality
of the OpenFunction, while also presenting the relationship between the OpenFunction
and its dependent components in a more abstract and friendly way to the user.

## Goals

- A more elegant way to display logs and progress

  It can collect logs and events of OpenFunction's resources in kubernetes
  and display them after correlation processing.

- Targeted resource manipulation

  It will simplify the parameters so that functions can be easily constructed.
  And it will follow the business model and operate on the OpenFunction and its
  associated resources uniformly, not just on the OpenFunction itself.

## Proposal

In this proposal I will named the command line tool for OpenFunction `fn`.

`fn` needs to manage and control the following resources:

- **build**, the function's builder, which will manage the information used to describe the process for building function.
- **serving**, the function's runner, which will manage the information used to describe the process for serving function.
- **function**, function's controller for stringing together the above resources and presenting the final function lifecycle management capabilities.

### Usage

[fn](#fn), provides management for Function.

[fn build](#fn-build), provides management for Build.

[fn serving](#fn-serving), provides management for Serving.

#### fn build

> build can be shortened to `fb` while contains the following sub commands:
>
> - [fn build create](#fn-build-create)
> - [fn build delete](#fn-build-delete)
> - [fn build update](#fn-build-update)
> - [fn build list](#fn-build-list)
> - [fn build describe](#fn-build-describe)


##### fn build create

> fn build create [OPTIONS]

***OPTIONS:***

    ```
      -b, --builder strings               the builder image
          --dry-run                       preview Build without running it
      -f, --filename string               local or remote file name containing a Build definition to create a Build
      -h, --help                          help for create
          --name strings                  the Build name
      -n, --namespace strings             the Namespace name (default: default)
          --output string                 format of Build dry-run (yaml or json)
          --params stringArray            the additional parameters for builder if needed
          --registry-account stringArray  the Secret which contains the login information for access to the mirror repository
          --registry-url string           the mirror repository address
          --registry-username string      alternative to account, requires --log-password
          --registry-password string      alternative to account, requires --log-username
      -s, --src-repo strings              the source code repository
          --src-name strings              the name of source code directory
    ```
    
***Example:***
      
    ```
    fn build create --name build_demo --namespace default 
        --builder "openfunction/gcp-builder:v1" \
        --params GOOGLE_FUNCTION_TARGET:"HelloWorld" \
        --params GOOGLE_FUNCTION_SIGNATURE_TYPE:"http" \
        --src-repo "https://github.com/OpenFunction/function-samples.git" \
        --src-name "hello-world-go" \
        --registry-url "https://index.docker.io/v1/" \
        --registry-username lamiar --registry-password *****
    ```

##### fn build delete

> fn build delete [OPTIONS] Name

***OPTIONS:***

    ```
          --all                           delete all Builds in a namespace (default: false)
          --dry-run                       preview Build without running it
      -n, --namespace strings             the Namespace name (default: default)
      -f, --force string                  whether to force deletion (default: false)
      -h, --help                          help for delete
          --output string                 format of Build dry-run (yaml or json)
    ```

***Example:***

    ```
    fn build delete build_demo
    ```

##### fn build update

> fn build update [OPTIONS] Name

***OPTIONS:***

    ```
          --dry-run                       preview Build without running it
      -h, --help                          help for update
          --output string                 format of Build dry-run (yaml or json)
      -n, --namespace strings             the Namespace name (default: default)
      -b, --builder strings               the builder image
          --params stringArray            the additional parameters for builder if needed
      -s, --src-repo strings              the source code repository
          --src-name strings              the name of source code directory
          --registry-url string           the mirror repository address
          --registry-account stringArray  the Secret which contains the login information for access to the mirror repository
          --registry-username string      alternative to account, requires --log-password
          --registry-password string      alternative to account, requires --log-username
    ```    

***Example:***

  ```
  fn build update --namespace default 
      --builder "openfunction/gcp-builder:v1" \
      --params GOOGLE_FUNCTION_TARGET:"HelloWorld" \
      --params GOOGLE_FUNCTION_SIGNATURE_TYPE:"http" \
      --src-repo "https://github.com/OpenFunction/function-samples.git" \
      --src-name "hello-world-go" \
      --registry-url "https://index.docker.io/v1/" \
      --registry-username lamiar --registry-password ***** \
      build_demo
  ```

##### fn build list

> fn build list [OPTIONS]

***OPTIONS:***

    ```
      -n, --namespace strings             the Namespace name (default: default)
      -l  --label stringArray             the labels for matching
      -h, --help                          help for list
          --output string                 format of Build dry-run (yaml or json)
    ```

***Example:***

    ```
    fn build list -l name:demo -n default
    ```

##### fn build describe

> fn build describe [OPTIONS] Name

***OPTIONS:***

    ```
      -n, --namespace strings             the Namespace name (default: default)
      -h, --help                          help for describe
          --output string                 format of Build dry-run (yaml or json)
    ```

***Example:***

    ```
    fn build describe --namespace ns1 build_demo
    ```

[Back to Usage :arrow_up_small:](#usage)

#### fn serving

> serving can be shortened to `fs` while contains the following sub commands:
>
> - [fn serving create](#fn-serving-create)
> - [fn serving delete](#fn-serving-delete)
> - [fn serving update](#fn-serving-update)
> - [fn serving list](#fn-serving-list)
> - [fn serving describe](#fn-serving-describe)

##### fn serving create

> fn serving create [OPTIONS]

***OPTIONS:***

    ```
      -r, --runtime strings               the Serving runtime, "Knative", "KEDA", e.g.
          --dry-run                       preview Serving without running it
      -f, --filename string               local or remote file name containing a Serving definition to create a Serving
      -h, --help                          help for create
          --name strings                  the Serving name
      -n, --namespace strings             the Namespace name (default: default)
          --output string                 format of Serving dry-run (yaml or json)
    ```

***Example:***

    ```
    fn serving create --name serving_demo --namespace default --runtime "Knative"
    ```

##### fn serving delete

> fn serving delete [OPTIONS] Name

***OPTIONS:***

    ```
          --all                           delete all Servings in a namespace (default: false)
          --dry-run                       preview Serving without running it
      -n, --namespace strings             the Namespace name (default: default)
      -f, --force string                  whether to force deletion (default: false)
      -h, --help                          help for delete
          --output string                 format of Serving dry-run (yaml or json)
    ```

***Example:***

    ```
    fn serving delete serving_demo
    ```

##### fn serving update

> fn serving update [OPTIONS] Name

***OPTIONS:***

    ```
      -r, --runtime strings               the Serving runtime, "Knative", "KEDA", e.g.
          --dry-run                       preview Serving without running it
      -f, --filename string               local or remote file name containing a Serving definition to create a Serving
      -h, --help                          help for create
      -n, --namespace strings             the Namespace name (default: default)
          --output string                 format of Serving dry-run (yaml or json)
    ```

***Example:***

    ```
    fn serving update --runtime "KEDA" serving_demo
    ```

##### fn serving list

> fn serving list [OPTIONS]

***OPTIONS:***

    ```
      -n, --namespace strings             the Namespace name (default: default)
      -l  --label stringArray             the labels for matching
      -h, --help                          help for list
          --output string                 format of Serving dry-run (yaml or json)
    ```

***Example:***

    ```
    fn serving list -l name:demo -n default
    ```

##### fn serving describe

> fn serving describe [OPTIONS] Name

***OPTIONS:***

    ```
      -n, --namespace strings             the Namespace name (default: default)
      -h, --help                          help for describe
          --output string                 format of Serving dry-run (yaml or json)
    ```

***Example:***

    ```
    fn serving describe --namespace ns1 serving_demo
    ```

[Back to Usage :arrow_up_small:](#usage)

#### fn

> function can be carried by `fn` directly while contains the following sub commands:
>
> - [fn create](#fn-create)
> - [fn delete](#fn-delete)
> - [fn update](#fn-update)
> - [fn list](#fn-list)
> - [fn describe](#fn-describe)

##### fn create

> fn create [OPTIONS]

***OPTIONS:***

    ```          
          --dry-run                       preview Function without running it
      -f, --filename string               local or remote file name containing a Function definition to create a Function
      -h, --help                          help for create
          --name strings                  the Function name
      -n, --namespace strings             the Namespace name (default: default)
          --output string                 format of Serving dry-run (yaml or json)
          --image string                  the target Function container image
          --version string                the Funtion version
          --build string                  name of the Build to be used
          --serving string                name of the Serving to be used
    ```

***Example:***

    ```
    fn create --name function_demo --namespace default \
        --version "1.0.0" --image "openfunction/sample-go-func:latest" \
        --build build_demo --serving serving_demo
    ```

##### fn delete

> fn delete [OPTIONS] Name

***OPTIONS:***

    ```
          --all                           delete all Functions in a namespace (default: false)
          --dry-run                       preview Function without running it
      -n, --namespace strings             the Namespace name (default: default)
      -f, --force string                  whether to force deletion (default: false)
      -h, --help                          help for delete
          --output string                 format of Function dry-run (yaml or json)
    ```

***Example:***

    ```
    fn delete function_demo
    ```

##### fn update

> fn update [OPTIONS] Name

***OPTIONS:***

    ```          
          --dry-run                       preview Function without running it
      -f, --filename string               local or remote file name containing a Function definition to create a Function
      -h, --help                          help for create
      -n, --namespace strings             the Namespace name (default: default)
          --output string                 format of Serving dry-run (yaml or json)
          --image string                  the target Function container image
          --version string                the Funtion version
          --build string                  name of the Build to be used
          --serving string                name of the Serving to be used
    ```

***Example:***

    ```
    fn update --namespace default --version "2.0.0" \
        --build build_demo_2 --serving serving_demo_2
        function_demo
    ```

##### fn list

> fn list [OPTIONS]

***OPTIONS:***

    ```
      -n, --namespace strings             the Namespace name (default: default)
      -l  --label stringArray             the labels for matching
      -h, --help                          help for list
          --output string                 format of Function dry-run (yaml or json)
    ```

***Example:***

    ```
    fn list -l name:demo -n default
    ```

##### fn describe

> fn describe [OPTIONS] Name

***OPTIONS:***

    ```
      -n, --namespace strings             the Namespace name (default: default)
      -h, --help                          help for describe
          --output string                 format of Function dry-run (yaml or json)
    ```

***Example:***

    ```
    fn describe --namespace ns1 function_demo
    ```

[Back to Usage :arrow_up_small:](#usage)
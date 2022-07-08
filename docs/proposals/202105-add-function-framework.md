## Motivation

In order to reduce the cost of learning the function specification when using OpenFunction, we need to add a mechanism to implement the conversion from user code to the main function in OpenFunction.

## Goals

Development of function conversion suites for different programming languages.

- Main function template

  During the build process of OpenFunction, the builder renders the user code using the main function template, based on which the mian function in the app image is generated.

- Function handler library

  Used to register different types of functions to the HTTP server to provide terminal access.

- Function Context

  A standard context structure to pass function semantics.

## Proposal

In short the goal is to make a generic main function for requests that come in through the function serving url. Of all the steps contained in this main function, one step will be used to associate the code submitted by the user, and the rest will be used to do common works (such as handling request references, handling context, handling event sources, handling exceptions, handling ports, etc.).

### Main function template

> Take the go language as an example

The user code is as follows.

```go
package hello

import (
    "fmt"
    "net/http"
)

func HelloWorld(w http.ResponseWriter, r *http.Request) {
    fmt.Fprint(w, "Hello, World!\n")
}
```

In order for this code to run in a serverless framework (such as Knative), the user would need to modify it to:

```go
package main

import (
    "fmt"
    "log"
    "net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprint(w, "Hello, World!\n")
}

func main() {
    http.HandleFunc("/", handler)
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
```

Using this as a basis, a simple main function template can be made.

```go
package main

import (
    "context"
    userFunction "{{.Package.Name}}"
    "log"
    "os"
)

func main() {
    http.HandleFunc("/", userFunction.{{.Package.Target}})
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
```

The user code is made into a Package, and then poured into the main function template for reference.

Based on the original code, where `.Package.Name` "somepath/hello" and `.Package.Target` is "HelloWorld", the rendered main function is as follows

```go
package main

import (
    "context"
    userFunction "user.function/hello"
    "log"
    "os"
)

func main() {
    http.HandleFunc("/", userFunction.HelloWorld)
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
```

### Function handler library

> Take the go language as an example

#### Register

In the above example, the function is of type HTTP, i.e. the user's code is only doing HTTP request handling. So the main function will fail when the user code is doing the handling of CloudEvent requests.

Here is a CloudEvent function:

```go
package hello

import (
    "context"
    "fmt"
    cloudevents "github.com/cloudevents/sdk-go/v2"
)

func HelloWorld(c context.Context, event cloudevents.Event) {
    fmt.Printf("%s\n", event.Data())
}
```

Processing one template to one function type will inevitably lead to an increase in the number of templates and the amount of work required to maintain them. So after having function conversion templates, we also need a library for handling different function types and other related works.

The features of this library are:

1. Determine the type of user code, such as HTTP or CloudEvent or other types (requires OpenFunction support first)
2. Register the function to the "/" path of the serving url
3. Convert the content of the request into a content format that the user code can receive
4. Serve the function on target port and listen

In the following example, I added a new registrar `RegisterHTTPFunction()` for the HTTP function, and an another registrar `RegisterCloudEventFunction()` for the CloudEvent function, and also added the serve function `Start()`.

```go
func RegisterHTTPFunction(ctx context.Context, pattern string, fn func(http.ResponseWriter, *http.Request)) error {
    http.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
        defer recoverPanicHTTP(w, "Function panic")
        fn(w, r)
    })
    return nil
}

func RegisterCloudEventFunction(ctx context.Context, pattern string, fn func(context.Context, cloudevents.Event)) error {
    p, err := cloudevents.NewHTTP()
    if err != nil {
        return fmt.Errorf("failed to create protocol: %v", err)
    }

    handleFn, err := cloudevents.NewHTTPReceiveHandler(ctx, p, fn)

    if err != nil {
        return fmt.Errorf("failed to create handler: %v", err)
    }

    http.Handle(pattern, handleFn)
    return nil
}

func Start(port string) error {
    log.Printf("Function serving: listening on port %s", port)
    log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
    return nil
}
```

Based on the above three functions, the main function template can be rewritten as follows

> `.Package.Name` is "somepath/hello" and `.Package.Target` is "HelloWorld".

```go
package main

import (
    "context"
    userFunction "{{.Package.Name}}" 
    functionFramework "example.com/hello/pkg/offf-go"
    "fmt"
    cloudevents "github.com/cloudevents/sdk-go/v2"
    "log"
    "net/http"
    "os"
)

func register(fn interface{}) error {
    ctx := context.Background()
    if fnHTTP, ok := fn.(func (http.ResponseWriter, *http.Request)); ok {
        if err := functionFramework.RegisterHTTPFunction(ctx, "/", fnHTTP); err != nil {
            return fmt.Errorf("Function failed to register: %v\n", err)
        }
    } else if fnCloudEvent, ok := fn.(func (context.Context, cloudevents.Event)); ok {
        if err := functionFramework.RegisterCloudEventFunction(ctx, "/", fnCloudEvent); err != nil {
            return fmt.Errorf("Function failed to register: %v\n", err)
        }
    }
    return nil
}

func main() {
    if err := register(userFunction.{{.Package.Target}}); err != nil {
        log.Fatalf("Failed to register: %v\n", err)
    }

    // Use PORT environment variable, or default to 8080.
    port := "8080"
    if envPort := os.Getenv("PORT"); envPort != "" {
       port = envPort
    }
    if err := functionFramework.Start(port); err != nil {
       log.Fatalf("Failed to start: %v\n", err)
    }
}
```

#### Function Context

Use the context structure to relate the definition of function crd and further provide the relevant tool suite to enable user to develop the code in a friendly, abstract form.

Here is an example. The user wants its function to fetch data from the input source, process it, and then send the result to the output target. Note that there may be multiple outputs.

```go
func BindingsFunction(w http.ResponseWriter, r *http.Request) {
    content, err := io.ReadAll(r.Body)

    type Data struct {
        OrderId int `json:"order_id"`
        Content string `json:"content"`
    }
    type payload struct {
        Data *Data `json:"data"`
        Operation string `json:"operation"`
    }

    n := 0

    for {
        n++
        p := &payload{}
        p.Data = &Data{
            OrderId: n,
            Content: content.(string),
        }
        p.Operation = "create"
        time.Sleep(1 * time.Second)

        _, err = http.Post("http://output.target", "application/json", strings.NewReader(string(body)))
        if err != nil {
            fmt.Errorf("Error: %v\n", err)
        }
    }

    if err != nil {
        w.WriteHeader(500)
    } else {
        w.WriteHeader(200)
    }
}
```

The role of the Function Context is to pass information about the resources reconcile by the function crd. For example, when you use Dapr, the context can associate some properties of the bindings component to your code.

The following is an example of the structure of a Function Context.

```go
type Output struct {
    Url string `json:"url"`
    Method HttpMethod `json:"method"`
}

type Input struct {
    Enabled *bool `json:"enabled"`
    Url string `json:"url"`
}

type Outputs struct {
    Enabled *bool `json:"enabled"`
    OutputObjects map[string]*Output `json:"output_objects"`
}

type Response struct {
    Code HttpResponseCode
}

type OpenFunctionContext struct {
    Name string `json:"name"`
    Version string `json:"version"`
    RequestID string `json:"request_id,omitempty"`
    Input *Input `json:"input,omitempty"`
    Outputs *Outputs `json:"outputs,omitempty"`
    // Runtime, Knative or Dapr
    Runtime Runtime `json:"runtime"`
    Response *Response `json:"response"`
}
```

And you can do the operations of getting data, sending data, etc. in the way of SDK without caring about the runtime implementation.

```go
type OpenFunctionContextInterface interface {
    SendTo(data interface{}, outputName string) error
    GetInput(r *http.Request) (interface{}, error)
}

func (ctx *OpenFunctionContext) SendTo(data interface{}, outputName string) error {
    send data to output
}

func (ctx *OpenFunctionContext) GetInput(r *http.Request) (interface{}, error) {
    get data from input
}
```

## Example

Here is a complete example with the user code and main function.

### User code

```go
func BindingsFunction(ctx *functionframeworks.OpenFunctionContext, r *http.Request) int {
    // ctx.GetInput is equal to "content, err := io.ReadAll(r.Body)"
    content, err := ctx.GetInput(r)

    type Data struct {
        OrderId int `json:"order_id"`
        Content string `json:"content"`
    }
    type payload struct {
        Data *Data `json:"data"`
        Operation string `json:"operation"`
    }

    n := 0

    for {
        n++
        p := &payload{}
        p.Data = &Data{
            OrderId: n,
            Content: content.(string),
        }
        p.Operation = "create"
        time.Sleep(1 * time.Second)
				
        // OUTPUT1 is the name of the bindings component in the function crd spec
        err := ctx.SendTo(p, "OUTPUT1")
        if err != nil {
            fmt.Errorf("Error: %v\n", err)
        }
    }

    if err != nil {
        return 500
    } else {
        return 200
    }
}

```

### Main function

```go
package main

import (
    "context"
    "github.com/OpenFunction/functions-framework-go/functionframeworks"
    userfunction "github.com/OpenFunction/functions-framework-go/testdata/demo"
    cloudevents "github.com/cloudevents/sdk-go/v2"
    "log"
    "net/http"
    "os"
)

func register(fn interface{}) error {
    ctx := context.Background()
    if fnHTTP, ok := fn.(func (http.ResponseWriter, *http.Request)); ok {
        if err := functionframeworks.RegisterHTTPFunction(ctx, fnHTTP); err != nil {
            return err
        }
    } else if fnCloudEvent, ok := fn.(func (context.Context, cloudevents.Event)); ok {
        if err := functionframeworks.RegisterCloudEventFunction(ctx, fnCloudEvent); err != nil {
            return err
        }
    } else if fnOpenFunction, ok := fn.(func (*functionframeworks.OpenFunctionContext, *http.Request) int); ok {
        if err := functionframeworks.RegisterOpenFunction(fnOpenFunction); err != nil {
            return err
        }
    }
    return nil
}

func main() {
    if err := register(userfunction.BindingsFunction); err != nil {
        log.Fatalf("Failed to register: %v\n", err)
    }

    port := "3000"
    if envPort := os.Getenv("PORT"); envPort != "" {
        port = envPort
    }

    if err := functionframeworks.Start(port); err != nil {
        log.Fatalf("Failed to start: %v\n", err)
    }
}

```

## Action Items

- Create a new repository called function-framework to store the specification of the function framework, as well as the index of associated items and content
- Define Input and Output resource specifications
- For different programming languages you need to complete.
    - Create a new repository called function-framework-{language-name} to store the function framework library for the corresponding programming language
    - Improve the function framework buildpack in builder
    - Write samples of different function types



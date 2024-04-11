# Support Dapr State management

## Motivation

Currently, OpenFunction supports Dapr [pub/sub](https://docs.dapr.io/reference/components-reference/supported-pubsub/) and [bindings](https://docs.dapr.io/reference/components-reference/supported-pubsub/) building blocks, [state management](https://docs.dapr.io/reference/components-reference/supported-state-stores/) is another building block that is useful for stateful functions, using a state store component, you can build stateful, long-running functions that save and retrieve their state.

## Goals

- You can define states stores in Function CR and OpenFunction will manage the corresponding Dapr components
- Your functions can use a simple encapsulated Dapr’s state management API to save, read, and query key/value pairs in the defined state stores

## Proposal

### OpenFunction

Add `states` field to the Function CRD:

```go
// Configurations of dapr bindings components.
// +optional
Bindings map[string]*componentsv1alpha1.ComponentSpec `json:"bindings,omitempty"`
// Configurations of dapr pubsub components.
// +optional
Pubsub map[string]*componentsv1alpha1.ComponentSpec `json:"pubsub,omitempty"`
// Configurations of dapr state components.
// +optional
States map[string]*componentsv1alpha1.ComponentSpec `json:"states,omitempty"`
```

When you define `spec.serving.states` in Function CR, the controller will do the following:

- When user does not define "openfunction.io/enable-dapr" in `spec.serving.annotations`, the value of this annotation will be treated as "true"
- Manage the corresponding Dapr components
- Generate states relevant context to the `FUNC_CONTEXT` env var

```yaml
apiVersion: core.openfunction.io/v1beta2
kind: Function
metadata:
  name: state-function-sample
spec:
  version: "v2.0.0"
  image: "openfunctiondev/state-func-go:v1"
  imageCredentials:
    name: push-secret
  port: 8080 # default to 8080
  serving:
    template:
      containers:
        - name: function # DO NOT change this
          imagePullPolicy: IfNotPresent
    runtime: "knative"
    states:
      - mysql:
        type: state.mysql
        version: v1
        metadata:
        - name: connectionString
          value: "<CONNECTION STRING>"
        - name: schemaName
          value: "<SCHEMA NAME>"
        - name: tableName
          value: "<TABLE NAME>"
        - name: timeoutInSeconds
          value: "30"
        - name: pemPath # Required if pemContents not provided. Path to pem file.
          value: "<PEM PATH>"
        - name: pemContents # Required if pemPath not provided. Pem value.
          value: "<PEM CONTENTS>"
```

### functions-framework

#### FUNC_CONTEXT

The implementation of each language of functions-framework needs to handle the `states` relevant context in `FUNC_CONTEXT`


> The following API relevant adjustments take functions-framework-go as an example, and the implementation of functions-framework in each language can be implemented according to the language itself

#### Pubsub API

```go
func (ctx *FunctionContext) PublishEvent(outputName string, data interface{}, opts ...PublishEventOption) error
```

#### Bindings API

```go
func (ctx *FunctionContext) InvokeBinding(outputName string, data []byte) (*BindingEvent, error)
```

#### State management API

##### Transaction

- ExecuteStateTransaction

```go
func (ctx *FunctionContext) ExecuteStateTransaction(storeName string, meta map[string]string, ops []*StateOperation) error
```

##### Save
- SaveState

```go
func (ctx *FunctionContext) SaveState(storeName, key string, data []byte, meta map[string]string, so ...StateOption) error
```
- SaveStateWithETag

```go
func (ctx *FunctionContext) SaveStateWithETag(storeName, key string, data []byte, etag string, meta map[string]string, so ...StateOption) error
```
- SaveBulkState

```go
func (ctx *FunctionContext) SaveBulkState(storeName string, items ...*SetStateItem) error
```

##### Get

- GetState

```go
func (ctx *FunctionContext) GetState(storeName, key string, meta map[string]string) (item *StateItem, err error)
```
- GetStateWithConsistency

```go
func (ctx *FunctionContext) GetStateWithConsistency(storeName, key string, meta map[string]string, sc StateConsistency) (*StateItem, error)
```

- GetBulkState

```go
func (ctx *FunctionContext) GetBulkState(storeName string, keys []string, meta map[string]string, parallelism int32) ([]*BulkStateItem, error)
```

##### Query

- QueryState

```go
func (ctx *FunctionContext) QueryState(storeName, query string, meta map[string]string) (*QueryResponse, error)
```

#### Delete

- DeleteState

```go
func (ctx *FunctionContext) DeleteState(storeName, key string, meta map[string]string) error
```

- DeleteStateWithETag

```go
func (ctx *FunctionContext) DeleteStateWithETag(storeName, key string, etag *ETag, meta map[string]string, opts *StateOptions) error
```

- DeleteBulkState

```go
func (ctx *FunctionContext) DeleteBulkState(storeName string, keys []string, meta map[string]string) error
```

- DeleteBulkStateItems

```go
func (ctx *FunctionContext) DeleteBulkStateItems(storeName string, items []*DeleteStateItem) error
```

## Action items

- Adjust OpenFunction CRD and Controller
- Adjust functions-framework of each language
- Add samples
- Add docs

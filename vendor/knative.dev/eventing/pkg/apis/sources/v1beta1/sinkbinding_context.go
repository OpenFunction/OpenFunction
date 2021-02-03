/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"context"

	"knative.dev/pkg/apis"
	"knative.dev/pkg/resolver"
)

// sinkURIKey is used as the key for associating information
// with a context.Context.
type sinkURIKey struct{}
type resolverKey struct{}

// WithSinkURI notes on the context for binding that the resolved SinkURI
// is the provided apis.URL.
func WithSinkURI(ctx context.Context, uri *apis.URL) context.Context {
	return context.WithValue(ctx, sinkURIKey{}, uri)
}

func WithURIResolver(ctx context.Context, resolver *resolver.URIResolver) context.Context {
	return context.WithValue(ctx, resolverKey{}, resolver)
}

// GetSinkURI accesses the apis.URL for the Sink URI that has been associated
// with this context.
func GetSinkURI(ctx context.Context) *apis.URL {
	value := ctx.Value(sinkURIKey{})
	if value == nil {
		return nil
	}
	return value.(*apis.URL)
}

func GetURIResolver(ctx context.Context) *resolver.URIResolver {
	value := ctx.Value(resolverKey{})
	if value == nil {
		return nil
	}
	return value.(*resolver.URIResolver)
}

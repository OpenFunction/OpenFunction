/*
Copyright 2022 The OpenFunction Authors.

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

package events

import (
	"context"
	"reflect"
	"testing"

	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	"github.com/go-logr/logr"
	log "github.com/go-logr/logr/testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	kservingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	ofcore "github.com/openfunction/apis/core/v1beta1"
	ofevent "github.com/openfunction/apis/events/v1alpha1"
)

func Test_createSinkComponent(t *testing.T) {
	type args struct {
		ctx      context.Context
		c        client.Client
		log      logr.Logger
		resource client.Object
		sink     *ofevent.SinkSpec
	}

	uri := "http://test"
	addressType := ofcore.InternalAddressType
	resource := &ofevent.EventSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}

	newSinkSpecFunc := func(t *testing.T, url string) *componentsv1alpha1.ComponentSpec {
		var spec componentsv1alpha1.ComponentSpec
		specMap := map[string]interface{}{
			"version": "v1",
			"type":    "bindings.http",
			"metadata": []map[string]string{
				{"name": "url", "value": url},
			},
		}
		specBytes, err := json.Marshal(specMap)
		if err != nil {
			t.Error(err)
			return nil
		}
		if err = json.Unmarshal(specBytes, &spec); err != nil {
			t.Error(err)
			return nil
		}
		return &spec
	}

	newServiceStatusFunc := func(t *testing.T, url string) *kservingv1.ServiceStatus {
		var status kservingv1.ServiceStatus
		statusMap := map[string]interface{}{
			"url": url,
		}

		statusBytes, err := json.Marshal(statusMap)
		if err != nil {
			t.Error(err)
			return nil
		}
		if err = json.Unmarshal(statusBytes, &status); err != nil {
			t.Error(err)
			return nil
		}
		return &status
	}

	newKnativeScheme := func(t *testing.T) *runtime.Scheme {
		scheme := runtime.NewScheme()
		err := kservingv1.AddToScheme(scheme)
		if err != nil {
			t.Error(err)
		}
		return scheme
	}

	newOfScheme := func(t *testing.T) *runtime.Scheme {
		scheme := runtime.NewScheme()
		err := ofcore.AddToScheme(scheme)
		if err != nil {
			t.Error(err)
		}
		return scheme
	}

	tests := []struct {
		name    string
		args    args
		want    *componentsv1alpha1.Component
		wantErr bool
	}{
		{
			name: "Use uri",
			args: args{
				ctx: context.Background(),
				c:   nil,
				log: &log.TestLogger{
					T: t,
				},
				resource: resource,
				sink: &ofevent.SinkSpec{
					Uri: &uri,
				},
			},
			want: &componentsv1alpha1.Component{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ts-test-test",
					Namespace: "test",
				},
				Spec: *newSinkSpecFunc(t, "http://test"),
			},
			wantErr: false,
		},
		{
			name: "Ref knative",
			args: args{
				ctx: context.Background(),
				c: fake.NewClientBuilder().WithScheme(newKnativeScheme(t)).WithRuntimeObjects(&kservingv1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "service",
						Namespace: "test",
					},
					Status: *newServiceStatusFunc(t, "http://test-service"),
				}).Build(),
				log: &log.TestLogger{
					T: t,
				},
				resource: resource,
				sink: &ofevent.SinkSpec{
					Ref: &ofevent.Reference{
						Kind:       "Service",
						APIVersion: kservingv1.SchemeGroupVersion.String(),
						Namespace:  "test",
						Name:       "service",
					},
				},
			},
			want: &componentsv1alpha1.Component{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ts-test-test",
					Namespace: "test",
				},
				Spec: *newSinkSpecFunc(t, "http://test-service"),
			},
			wantErr: false,
		},
		{
			name: "ref openfunction",
			args: args{
				ctx: context.Background(),
				c: fake.NewClientBuilder().WithScheme(newOfScheme(t)).WithRuntimeObjects(&ofcore.Function{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "function",
						Namespace: "test",
					},
					Status: ofcore.FunctionStatus{
						Addresses: []ofcore.FunctionAddress{{Type: &addressType, Value: "http://test-of"}},
					},
				}).Build(),
				log: &log.TestLogger{
					T: t,
				},
				resource: resource,
				sink: &ofevent.SinkSpec{
					Ref: &ofevent.Reference{
						Kind:       "Function",
						APIVersion: ofcore.SchemeGroupVersion.String(),
						Namespace:  "test",
						Name:       "function",
					},
				},
			},
			want: &componentsv1alpha1.Component{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ts-test-test",
					Namespace: "test",
				},
				Spec: *newSinkSpecFunc(t, "http://test-of"),
			},
			wantErr: false,
		},
		{
			name: "Set both",
			args: args{
				ctx: context.Background(),
				c: fake.NewClientBuilder().WithScheme(newKnativeScheme(t)).WithRuntimeObjects(&kservingv1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Status: *newServiceStatusFunc(t, "http://test"),
				}).Build(),
				log: &log.TestLogger{
					T: t,
				},
				resource: resource,
				sink: &ofevent.SinkSpec{
					Ref: &ofevent.Reference{
						Kind:       "Service",
						APIVersion: "serving.knative.dev/v1",
						Namespace:  "test",
						Name:       "test-ref",
					},
					Uri: &uri,
				},
			},
			want: &componentsv1alpha1.Component{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ts-test-test",
					Namespace: "test",
				},
				Spec: *newSinkSpecFunc(t, "http://test"),
			},
			wantErr: false,
		},
		{
			name: "Failed to find Knative Service",
			args: args{
				ctx: context.Background(),
				c: fake.NewClientBuilder().WithScheme(newKnativeScheme(t)).WithRuntimeObjects(&kservingv1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Status: *newServiceStatusFunc(t, "http://test-ref"),
				}).Build(),
				log: &log.TestLogger{
					T: t,
				},
				resource: resource,
				sink: &ofevent.SinkSpec{
					Ref: &ofevent.Reference{
						Kind:       "Service",
						APIVersion: "serving.knative.dev/v1",
						Namespace:  "test",
						Name:       "test-not-found",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Failed to find OpenFunction",
			args: args{
				ctx: context.Background(),
				c: fake.NewClientBuilder().WithScheme(newOfScheme(t)).WithRuntimeObjects(&ofcore.Function{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "function",
						Namespace: "test",
					},
					Status: ofcore.FunctionStatus{
						Addresses: []ofcore.FunctionAddress{{Type: &addressType, Value: "http://test-of"}},
					},
				}).Build(),
				log: &log.TestLogger{
					T: t,
				},
				resource: resource,
				sink: &ofevent.SinkSpec{
					Ref: &ofevent.Reference{
						Kind:       "Function",
						APIVersion: ofcore.SchemeGroupVersion.String(),
						Namespace:  "test",
						Name:       "test-not-found",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "unsupported reference",
			args: args{
				ctx: context.Background(),
				c:   nil,
				log: &log.TestLogger{
					T: t,
				},
				resource: resource,
				sink: &ofevent.SinkSpec{
					Ref: &ofevent.Reference{
						Kind:       "Function",
						APIVersion: "core.openfunction.io/test",
						Namespace:  "test",
						Name:       "test",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "None of them are set",
			args: args{
				ctx: context.Background(),
				log: &log.TestLogger{
					T: t,
				},
				sink: &ofevent.SinkSpec{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createSinkComponent(tt.args.ctx, tt.args.c, tt.args.log, tt.args.resource, tt.args.sink)
			if (err != nil) != tt.wantErr {
				t.Errorf("createSinkComponent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createSinkComponent() got = %v, want %v", got, tt.want)
			}
		})
	}
}

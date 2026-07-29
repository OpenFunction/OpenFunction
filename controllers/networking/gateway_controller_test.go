/*
Copyright 2026 The OpenFunction Authors.

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

package networking

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	networkingv1alpha1 "github.com/openfunction/apis/networking/v1alpha1"
)

func TestUpdateGatewayAnnotationsInitializesMissingAnnotations(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	if err := networkingv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add networking API to scheme: %v", err)
	}

	gateway := &networkingv1alpha1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Name: "gateway", Namespace: "default"},
		Spec:       networkingv1alpha1.GatewaySpec{Domain: "example.com"},
	}
	reconciler := &GatewayReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(gateway).Build(),
		ctx:    ctx,
	}

	if err := reconciler.updateGatewayAnnotations(gateway); err != nil {
		t.Fatalf("update gateway annotations: %v", err)
	}

	updated := &networkingv1alpha1.Gateway{}
	if err := reconciler.Get(ctx, client.ObjectKeyFromObject(gateway), updated); err != nil {
		t.Fatalf("get updated gateway: %v", err)
	}
	annotation, ok := updated.Annotations[networkingv1alpha1.GatewayConfigAnnotation]
	if !ok {
		t.Fatal("last-applied configuration annotation was not set")
	}

	var lastApplied networkingv1alpha1.Gateway
	if err := json.Unmarshal([]byte(annotation), &lastApplied); err != nil {
		t.Fatalf("unmarshal last-applied configuration: %v", err)
	}
	if lastApplied.Spec.Domain != gateway.Spec.Domain {
		t.Fatalf("last-applied domain = %q, want %q", lastApplied.Spec.Domain, gateway.Spec.Domain)
	}
}

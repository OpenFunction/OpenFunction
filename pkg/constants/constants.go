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

package constants

import (
	"sigs.k8s.io/gateway-api/apis/v1beta1"
)

const (
	FunctionLabel = "openfunction.io/function"

	CommonLabelVersion = "app.kubernetes.io/version"

	DefaultFunctionVersion = "latest"

	DefaultConfigMapName       = "openfunction-config"
	DefaultControllerNamespace = "openfunction"

	DefaultKnativeServingNamespace      = "knative-serving"
	DefaultKnativeServingFeaturesCMName = "config-features"

	DefaultKedaInterceptorProxyName = "keda-add-ons-http-interceptor-proxy"

	DefaultGatewayName             v1beta1.ObjectName   = "openfunction"
	DefaultGatewayNamespace        v1beta1.Namespace    = "openfunction"
	DefaultGatewayListenerPort     v1beta1.PortNumber   = 80
	DefaultGatewayListenerProtocol v1beta1.ProtocolType = "HTTP"
	DefaultFunctionServicePort     v1beta1.PortNumber   = 80
	DefaultFuncPort                v1beta1.PortNumber   = 8080
	DefaultInterceptorPort         v1beta1.PortNumber   = 8080

	WasmEdgeWorkloadRuntimeName = "wasmedge"
	WasmEdgeRuntimeClassName    = "openfunction-crun"
	WasmEdgeBuildStrategyName   = "wasmedge"
	WasmVariantAnnotation       = "module.wasm.image/variant"
	WasmCompatSmart             = "compat-smart"
)

#!/bin/bash

# Copyright 2022 The OpenFunction Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

all=false
with_cert_manager=false
with_shipwright=false
with_knative=false
with_gateway=false
with_openFuncAsync=false

# add help information
if [[ ${1} = "--help" ]] || [[ ${1} = "-h" ]]
then
  clear
  echo "-----------------------------------------------------------------------------------------------"
  echo "This shell script used to deploy OpenFunction components in your Cluster"
  echo "--all Will install cert_manager shipwright knative openFuncAsync gateway components"
  echo "--with-cert-manager Will install cert-manager component"
  echo "--with-shipwright Will install shipwright component"
  echo "--with-knative Will install knative component"
  echo "--with-gateway Will install gateway component"
  echo "--with-openFuncAsync Will install dapr and keda components"
  echo "-----------------------------------------------------------------------------------------------"
  exit 0
fi

if [ $? != 0 ]; then
  echo "Terminating..." >&2
  exit 1
fi

while test $# -gt 0; do
  case "${1}" in
  --all)
    all=true
    ;;
  --with-cert-manager)
    with_cert_manager=true
    ;;
  --with-shipwright)
    with_shipwright=true
    ;;
  --with-knative)
    with_knative=true
    ;;
  --with-gateway)
      with_gateway=true
      ;;
  --with-openFuncAsync)
    with_openFuncAsync=true
    ;;
  *)
    echo "Internal error!"
    exit 1
    ;;
  esac
  shift
done

if [ "${all}" = "true" ]; then
  with_cert_manager=true
  with_shipwright=true
  with_knative=true
  with_gateway=true
  with_openFuncAsync=true
fi

if [ "${with_cert_manager}" = "true" ]; then
  kubectl apply --filename https://github.com/jetstack/cert-manager/releases/download/v1.5.4/cert-manager.yaml
fi

if [ "${with_shipwright}" = "true" ]; then
  # Install Tekton pipeline
  kubectl apply --filename https://github.com/tektoncd/pipeline/releases/download/v0.28.1/release.yaml
  # Install the Shipwright deployment
  kubectl apply --filename https://github.com/shipwright-io/build/releases/download/v0.6.0/release.yaml
fi

if [ "${with_knative}" = "true" ]; then
  # Install the required custom resources
  kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.3.2/serving-crds.yaml
  # Install the core components of Serving
  kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.3.2/serving-core.yaml
  # Install a networking layer
  # Install the Knative Kourier controller
  kubectl apply -f https://github.com/knative/net-kourier/releases/download/knative-v1.3.0/kourier.yaml
  # To configure Knative Serving to use Contour by default
  kubectl patch configmap/config-network \
    --namespace knative-serving \
    --type merge \
    --patch '{"data":{"ingress.class":"kourier.ingress.networking.knative.dev"}}'
  # Configure DNS
  kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.3.2/serving-default-domain.yaml
fi

if [ "${with_gateway}" = "true" ]; then
  # Deploy the Gateway provisioner
  kubectl apply -f https://raw.githubusercontent.com/projectcontour/contour/v1.21.1/examples/render/contour-gateway-provisioner.yaml
  # Wait for Gateway provisioner to be ready
  while /bin/true; do
    admission_status=$(kubectl get deployment -n gateway-api gateway-api-admission-server -o jsonpath='{.status.conditions[?(@.type=="Available")].status}')
    if [ "$admission_status" == "True" ]; then
      echo "Contour Gateway provisioner is ready"
      break
    else
      sleep 1
      continue
    fi
  done
  sleep 10
  # Create a GatewayClass
  kubectl apply -f - <<EOF
  kind: GatewayClass
  apiVersion: gateway.networking.k8s.io/v1alpha2
  metadata:
    name: contour
  spec:
    controllerName: projectcontour.io/gateway-controller
EOF
  # Create a Gateway
  kubectl apply -f - <<EOF
  kind: Gateway
  apiVersion: gateway.networking.k8s.io/v1alpha2
  metadata:
    name: contour
    namespace: projectcontour
  spec:
    gatewayClassName: contour
    listeners:
      - name: http
        protocol: HTTP
        port: 80
        allowedRoutes:
          namespaces:
            from: All
EOF
fi

if [ "${with_openFuncAsync}" = "true" ]; then
  # Installs the latest Dapr CLI.
  wget -q https://raw.githubusercontent.com/dapr/cli/master/install/install.sh -O - | /bin/bash -s 1.4.0
  # Init dapr
  dapr init -k --runtime-version 1.3.1
  # Installs the latest release version
  kubectl apply -f https://github.com/kedacore/keda/releases/download/v2.4.0/keda-2.4.0.yaml
fi

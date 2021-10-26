#!/bin/bash

TEMP=$(getopt -o an:p --long all,with-shipwright,with-knative,with-openFuncAsync,poor-network -- "$@")

all=false
with_shipwright=false
with_knative=false
with_openFuncAsync=false
poor_network=false

if [ $? != 0 ]; then
  echo "Terminating..." >&2
  exit 1
fi

# Note the quotes around `$TEMP': they are essential!
eval set -- "$TEMP"

while true; do
  case "$1" in
  --all)
    all=true
    shift
    ;;
  --with-shipwright)
    with_shipwright=true
    shift
    ;;
  --with-knative)
    with_knative=true
    shift
    ;;
  --with-openFuncAsync)
    with_openFuncAsync=true
    shift
    ;;
  -p | --poor-network)
    poor_network=true
    shift
    ;;
  --)
    shift
    break
    ;;
  *)
    echo "Internal error!"
    exit 1
    ;;
  esac
done

if [ "$all" = "true" ]; then
  with_shipwright=true
  with_knative=true
  with_openFuncAsync=true
fi

if [ "$poor_network" = "false" ]; then
  kubectl apply --filename https://github.com/jetstack/cert-manager/releases/download/v1.5.4/cert-manager.yaml
else
  kubectl apply --filename https://openfunction.sh1a.qingstor.com/cert-manager/v1.5.4/cert-manager.yaml
fi

if [ "$with_shipwright" = "true" ]; then
  if [ "$poor_network" = "false" ]; then
    # Install Tekton pipeline
    kubectl apply --filename https://github.com/tektoncd/pipeline/releases/download/v0.28.1/release.yaml
    # Install the Shipwright deployment
    kubectl apply --filename https://github.com/shipwright-io/build/releases/download/v0.6.0/release.yaml
  else
    # Install Tekton pipeline
    kubectl apply --filename https://openfunction.sh1a.qingstor.com/tekton/pipeline/v0.28.1/release.yaml
    # Install the Shipwright deployment
    kubectl apply --filename https://openfunction.sh1a.qingstor.com/shipwright/v0.6.0/release.yaml
  fi
fi

if [ "$with_knative" = "true" ]; then
  if [ "$poor_network" = "false" ]; then
    # Install the required custom resources
    kubectl apply -f https://github.com/knative/serving/releases/download/v0.26.0/serving-crds.yaml
    # Install the core components of Serving
    kubectl apply -f https://github.com/knative/serving/releases/download/v0.26.0/serving-core.yaml
    # Install a networking layer
    # Install the Knative Kourier controller
    kubectl apply -f https://github.com/knative/net-kourier/releases/download/v0.26.0/kourier.yaml
    # To configure Knative Serving to use Kourier by default
    kubectl patch configmap/config-network \
      --namespace knative-serving \
      --type merge \
      --patch '{"data":{"ingress.class":"kourier.ingress.networking.knative.dev"}}'
    # Configure DNS
    kubectl apply -f https://github.com/knative/serving/releases/download/v0.26.0/serving-default-domain.yaml
  else
    # Install the required custom resources
    kubectl apply -f https://openfunction.sh1a.qingstor.com/knative/serving/v0.26.0/serving-crds.yaml
    # Install the core components of Serving
    kubectl apply -f https://openfunction.sh1a.qingstor.com/knative/serving/v0.26.0/serving-core.yaml
    # Install a networking layer
    # Install the Knative Kourier controller
    kubectl apply -f https://openfunction.sh1a.qingstor.com/knative/net-kourier/v0.26.0/kourier.yaml
    # To configure Knative Serving to use Kourier by default
    kubectl patch configmap/config-network \
      --namespace knative-serving \
      --type merge \
      --patch '{"data":{"ingress.class":"kourier.ingress.networking.knative.dev"}}'
    # Configure DNS
    kubectl apply -f https://openfunction.sh1a.qingstor.com/knative/serving/v0.26.0/serving-default-domain.yaml
  fi
fi

if [ "$with_openFuncAsync" = "true" ]; then
  if [ "$poor_network" = "false" ]; then
    # Installs the latest Dapr CLI.
    wget -q https://raw.githubusercontent.com/dapr/cli/master/install/install.sh -O - | /bin/bash -s 1.4.0
    # Init dapr
    dapr init -k --runtime-version 1.3.1
    # Installs the latest release version
    kubectl apply -f https://github.com/kedacore/keda/releases/download/v2.4.0/keda-2.4.0.yaml
   else
    # Installs the latest Dapr CLI.
    wget -q https://openfunction.sh1a.qingstor.com/dapr/install.sh -O - | /bin/bash -s 1.4.0
    # Init dapr
    dapr init -k --runtime-version 1.3.1
    # Installs the latest release version
    kubectl apply -f https://openfunction.sh1a.qingstor.com/keda/v2.4.0/keda-2.4.0.yaml
  fi
fi

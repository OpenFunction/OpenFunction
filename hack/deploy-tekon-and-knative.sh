#!/bin/bash

TEMP=$(getopt -o an:p --long poor-network,tekton-dashboard-nodeport: -- "$@")

poor_network=false
tekton_dashboard_nodeport=0

if [ $? != 0 ]; then
  echo "Terminating..." >&2
  exit 1
fi

# Note the quotes around `$TEMP': they are essential!
eval set -- "$TEMP"

while true; do
  case "$1" in
  -p | --poor-network)
    poor_network=true
    shift
    ;;
  -n | --tekton-dashboard-nodeport)
    tekton_dashboard_nodeport=$2
    shift 2
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

if [ "$poor_network" = "false" ]; then
  # Install Tekton pipeline
  kubectl apply --filename https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
  # Install Tekton triggers
  kubectl apply --filename https://storage.googleapis.com/tekton-releases/triggers/latest/release.yaml
  # Install Tekton Dashboard
  kubectl apply --filename https://github.com/tektoncd/dashboard/releases/latest/download/tekton-dashboard-release.yaml
  # Install the required custom resources
  kubectl apply -f https://github.com/knative/serving/releases/download/v0.22.0/serving-crds.yaml
  # Install the core components of Serving
  kubectl apply -f https://github.com/knative/serving/releases/download/v0.22.0/serving-core.yaml
  # Install a networking layer
  # Install the Knative Kourier controller
  kubectl apply -f https://github.com/knative/net-kourier/releases/download/v0.22.0/kourier.yaml
  # To configure Knative Serving to use Kourier by default
  kubectl patch configmap/config-network \
    --namespace knative-serving \
    --type merge \
    --patch '{"data":{"ingress.class":"kourier.ingress.networking.knative.dev"}}'
  # Configure DNS
  kubectl apply -f https://github.com/knative/serving/releases/download/v0.22.0/serving-default-domain.yaml
  # Install Knative Eventing
  # Install the required custom resource definitions (CRDs)
  #kubectl apply -f https://github.com/knative/eventing/releases/download/v0.22.0/eventing-crds.yaml
  # Install the core components of Eventing
  kubectl apply -f https://github.com/knative/eventing/releases/download/v0.22.0/eventing-core.yaml
else
  # Install Tekton pipeline
  kubectl apply --filename https://openfunction.sh1a.qingstor.com/tekton/pipeline/v0.23.0/release.yaml
  # Install Tekton triggers
  kubectl apply --filename https://openfunction.sh1a.qingstor.com/tekton/trigger/v0.13.0/release.yaml
  # Install Tekton Dashboard
  kubectl apply --filename https://openfunction.sh1a.qingstor.com/tekton/dashboard/v0.16.0/release.yaml
  # Install the required custom resources
  kubectl apply -f https://openfunction.sh1a.qingstor.com/knative/serving/v0.22.0/serving-crds.yaml
  # Install the core components of Serving
  kubectl apply -f https://openfunction.sh1a.qingstor.com/knative/serving/v0.22.0/serving-core.yaml
  # Install a networking layer
  # Install the Knative Kourier controller
  kubectl apply -f https://openfunction.sh1a.qingstor.com/knative/net-kourier/v0.22.0/kourier.yaml
  # To configure Knative Serving to use Kourier by default
  kubectl patch configmap/config-network \
    --namespace knative-serving \
    --type merge \
    --patch '{"data":{"ingress.class":"kourier.ingress.networking.knative.dev"}}'
  # Configure DNS
  kubectl apply -f https://openfunction.sh1a.qingstor.com/knative/serving/v0.22.0/serving-default-domain.yaml
  # Install the required custom resource definitions (CRDs)
  #kubectl apply -f https://openfunction.sh1a.qingstor.com/knative/eventing/v0.22.0/eventing-crds.yaml
  # Install the core components of Eventing
  kubectl apply -f https://openfunction.sh1a.qingstor.com/knative/eventing/v0.22.0/eventing-core.yaml
fi

patch=$(echo "{\"spec\":{\"ports\":[{\"name\":\"http\",\"nodePort\":$tekton_dashboard_nodeport,\"port\":9097,\"protocol\":\"TCP\",\"targetPort\":9097}],\"type\":\"NodePort\"}}")

if [ "$tekton_dashboard_nodeport" != "0" ]; then
  kubectl patch service/tekton-dashboard \
    --namespace tekton-pipelines \
    --type merge \
    --patch $patch
fi
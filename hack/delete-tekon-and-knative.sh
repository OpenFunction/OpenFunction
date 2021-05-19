#!/bin/bash

version=v0.23.0
if [ $1 ]; then
    version=$1
fi

kubectl delete --filename https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
# delete Tekton triggers
kubectl delete --filename https://storage.googleapis.com/tekton-releases/triggers/latest/release.yaml
# delete Tekton Dashboard
kubectl delete --filename https://github.com/tektoncd/dashboard/releases/latest/download/tekton-dashboard-release.yaml
# delete the required custom resources
kubectl delete -f https://github.com/knative/serving/releases/download/$version/serving-crds.yaml
# delete the core components of Serving
kubectl delete -f https://github.com/knative/serving/releases/download/$version/serving-core.yaml
# delete a networking layer
# delete the Knative Kourier controller
kubectl delete -f https://github.com/knative/net-kourier/releases/download/$version/kourier.yaml
# Configure DNS
kubectl delete -f https://github.com/knative/serving/releases/download/$version/serving-default-domain.yaml
# delete Knative Eventing
# delete the required custom resource definitions (CRDs)
kubectl delete -f https://github.com/knative/eventing/releases/download/$version/eventing-crds.yaml
# delete the core components of Eventing
kubectl delete -f https://github.com/knative/eventing/releases/download/$version/eventing-core.yaml

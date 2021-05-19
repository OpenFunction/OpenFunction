#!/bin/bash

version=v0.23.0
if [ $1 ]; then
    version=$1
fi

# delete Tekton pipeline
kubectl delete --filename https://openfunction.sh1a.qingstor.com/tekton/pipeline/${version}/release.yaml
# delete Tekton triggers
kubectl delete --filename https://openfunction.sh1a.qingstor.com/tekton/trigger/${version}/release.yaml
# delete Tekton Dashboard
kubectl delete --filename https://openfunction.sh1a.qingstor.com/tekton/dashboard/${version}/release.yaml
# delete the required custom resources
kubectl delete -f https://openfunction.sh1a.qingstor.com/knative/serving/${version}/serving-crds.yaml
# delete the core components of Serving
kubectl delete -f https://openfunction.sh1a.qingstor.com/knative/serving/${version}/serving-core.yaml
# delete a networking layer
# delete the Knative Kourier controller
kubectl delete -f https://openfunction.sh1a.qingstor.com/knative/net-kourier/${version}/kourier.yaml
# To configure Knative Serving to use Kourier by default
# Configure DNS
kubectl delete -f https://openfunction.sh1a.qingstor.com/knative/serving/${version}/serving-default-domain.yaml
# delete the required custom resource definitions (CRDs)
#kubectl delete -f https://openfunction.sh1a.qingstor.com/knative/eventing/v0.23.0/eventing-crds.yaml
# delete the core components of Eventing
kubectl delete -f https://openfunction.sh1a.qingstor.com/knative/eventing/${version}/eventing-core.yaml

#! /bin/bash

# Install dependent components
chmod a+x ./hack/deploy.sh
./hack/deploy.sh --all
# Remove the resources request to avoid Insufficient CPU error
kubectl patch deployments.apps -n tekton-pipelines tekton-pipelines-webhook -p '{"spec":{"template":{"spec":{"containers":[{"name":"webhook","resources":null}]}}}}'
kubectl patch deployments.apps -n knative-serving activator -p '{"spec":{"template":{"spec":{"containers":[{"name":"activator","resources":null}]}}}}'
kubectl patch deployments.apps -n knative-serving controller -p '{"spec":{"template":{"spec":{"containers":[{"name":"controller","resources":null}]}}}}'
kubectl patch deployments.apps -n knative-serving autoscaler -p '{"spec":{"template":{"spec":{"containers":[{"name":"autoscaler","resources":null}]}}}}'
kubectl patch deployments.apps -n knative-serving domainmapping-webhook -p '{"spec":{"template":{"spec":{"containers":[{"name":"domainmapping-webhook","resources":null}]}}}}'
kubectl patch deployments.apps -n knative-serving webhook -p '{"spec":{"template":{"spec":{"containers":[{"name":"webhook","resources":null}]}}}}'
kubectl patch deployments.apps -n keda keda-operator -p '{"spec":{"template":{"spec":{"containers":[{"name":"keda-operator","resources":null}]}}}}'
kubectl patch deployments.apps -n keda keda-metrics-apiserver -p '{"spec":{"template":{"spec":{"containers":[{"name":"keda-metrics-apiserver","resources":null}]}}}}'

# Install kafka
helm repo add strimzi https://strimzi.io/charts/
helm install kafka-operator -n default strimzi/strimzi-kafka-operator
kubectl apply -f config/samples/function-kafka-quick.yaml

# Install OpenFunction
kubectl create -f config/bundle.yaml
kubectl patch deployments.apps -n openfunction openfunction-controller-manager -p "{\"spec\":{\"template\":{\"spec\":{\"containers\":[{\"name\":\"openfunction\",\"image\":\"openfunctiondev/openfunction:$1\",\"resources\":null}]}}}}"

# Wait for all to be ready
sleep 60

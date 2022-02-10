#! /bin/bash

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

function knative_function() {
    kubectl apply -f config/samples/function-sample-serving-only.yaml
    kubectl proxy &

    while /bin/true; do
      url=$(kubectl get fn function-sample-serving-only -o jsonpath='{.status.url}')
      if [ -z "$url" ]; then
        sleep 1
        continue
      else
        echo "Function function-sample-serving-only is running"
        break
      fi
    done

    url="http://localhost:8001/api/v1/namespaces/ingress-nginx/services/ingress-nginx-controller:http/proxy/default/function-sample-serving-only"
    while /bin/true; do
      res=$(curl -I -m 10 -o /dev/null -s -w %{http_code}"\n" $url)
      if test "$res" = "200"; then
        echo "Knative function tested successfully!"
        kubectl delete -f config/samples/function-sample-serving-only.yaml
        break
      fi
      sleep 1
    done

    pkill -9 -f "kubectl proxy"
}

function knative_function_with_dapr() {
    kubectl apply -f config/samples/function-knative-with-dapr-serving-only.yaml
    kubectl proxy &

    while /bin/true; do
      url=$(kubectl get fn function-front -o jsonpath='{.status.url}')
      if [ -z "$url" ]; then
        sleep 1
        continue
      else
        echo "Function function-knative-with-dapr-serving-only is running"
        break
      fi
    done

    url="http://localhost:8001/api/v1/namespaces/ingress-nginx/services/ingress-nginx-controller:http/proxy/default/function-front"
    while /bin/true; do
      res=$(curl -m 10 -o /dev/null -s -w %{http_code}"\n" -d '{"message":"Awesome OpenFunction!"}' -H "Content-Type: application/json" -X POST $url)
      if test "$res" = "200"; then
        break
      fi
      sleep 1
    done

    while /bin/true; do
      kubectl logs $(kubectl get po -l openfunction.io/serving=$(kubectl get functions output-target -o jsonpath='{.status.serving.resourceRef}') \
        -o jsonpath='{.items[0].metadata.name}') function |grep "Awesome OpenFunction"
      if [ $? -ne 0 ]; then
        sleep 1
        continue
      else
        echo "Knative function with dapr tested successfully!"
        kubectl delete -f config/samples/function-knative-with-dapr-serving-only.yaml
        break
      fi
    done

    pkill -9 -f "kubectl proxy"
}

function async_function_with_bindings() {
    kubectl apply -f config/samples/function-bindings-sample-serving-only.yaml

    while /bin/true; do
      kubectl logs $(kubectl get po -l openfunction.io/serving=$(kubectl get functions output-target -o jsonpath='{.status.serving.resourceRef}') \
        -o jsonpath='{.items[0].metadata.name}') function |grep "Hello"
      if [ $? -ne 0 ]; then
        sleep 1
        continue
      else
        echo "Async function with bindings tested successfully!"
        kubectl delete -f config/samples/function-bindings-sample-serving-only.yaml
        break
      fi
    done
}

function function_with_plugins() {
    kubectl apply -f config/samples/function-with-plugins-serving-only.yaml

    while /bin/true; do
      kubectl logs $(kubectl get po -l openfunction.io/serving=$(kubectl get functions bindings-plugins -o jsonpath='{.status.serving.resourceRef}') \
        -o jsonpath='{.items[0].metadata.name}') function |grep "the sum is: 2"
      if [ $? -ne 0 ]; then
        sleep 1
        continue
      else
        echo "Function with plugins tested successfully!"
        kubectl delete -f config/samples/function-with-plugins-serving-only.yaml
        break
      fi
    done
}

function async_function_with_pubsub() {
    kubectl apply -f config/samples/function-pubsub-sample-serving-only.yaml

    while /bin/true; do
      rec=`kubectl logs $(kubectl get po -l openfunction.io/serving=$(kubectl get functions autoscaling-subscriber -o jsonpath='{.status.serving.resourceRef}') -o jsonpath='{.items[0].metadata.name}') function |grep "event - Data"|wc -l`
      if [ $rec -gt 0 ]; then
        break
      else
        sleep 1
        continue
      fi
    done

    while /bin/true; do
      replicas=`kubectl get deploy -l openfunction.io/serving=$(kubectl get functions autoscaling-subscriber -o jsonpath='{.status.serving.resourceRef}') -o jsonpath='{.items[0].spec.replicas}'`
      if [ $replicas -gt 1 ]; then
        echo "Async function with pubsub tested successfully!"
        kubectl delete -f config/samples/function-pubsub-sample-serving-only.yaml
        break
      else
        sleep 1
        continue
      fi
    done
}

case $1 in

  knative)
    knative_function
    ;;

  knative_dapr)
    knative_function_with_dapr
    ;;

  async_bindings)
    async_function_with_bindings
    ;;

  async_pubsub)
    async_function_with_pubsub
    ;;

  plugin)
    function_with_plugins
    ;;
esac

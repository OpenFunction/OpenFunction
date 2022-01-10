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

kubectl apply -f config/samples/function-sample-serving-only.yaml
kubectl apply -f config/samples/function-pubsub-sample-serving-only.yaml
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
    curl $url
    break
  fi
  sleep 1
done

while /bin/true; do
  state=$(kubectl get fn autoscaling-producer -o jsonpath='{.status.serving.state}')
  if test "$state" != "Running"; then
    sleep 1
    continue
  else
    echo "Function autoscaling-producer is running"
    break
  fi
done

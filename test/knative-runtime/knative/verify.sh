#!/bin/bash

#
# Copyright 2022 The OpenFunction Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

export KUBECONFIG=/tmp/e2e-k8s.config

MANIFESTS_PATH="./test/knative-runtime/knative/manifests.yaml"

if [ "${DOCKERHUB_REPO_PREFIX}" ]; then
  declare -a FUNCTIONS=("e2e-v1beta1-knative")
  for fn in "${FUNCTIONS[@]}";do
    IMAGE_NAME=${DOCKERHUB_REPO_PREFIX}/$(NAME=${fn} ./bin/yq e 'select(.metadata.name == env(NAME) ).spec.image' ${MANIFESTS_PATH}|awk -F\/ '{ print $2 }') NAME=${fn} ./bin/yq -i e 'select(.metadata.name == env(NAME)).spec.image |= env(IMAGE_NAME)' ${MANIFESTS_PATH}
  done
fi

kubectl port-forward -n ingress-nginx service/ingress-nginx-controller 8080:80 > /dev/null 2>&1 &

kubectl apply -f ${MANIFESTS_PATH} > /dev/null 2>&1

url=$(kubectl get fn e2e-v1beta1-knative -o jsonpath='{.status.url}')
if [ "$url" ]; then
  res=$(curl -H "Host: openfunction.io.svc" -I -m 10 -o /dev/null -s -w %{http_code}"\n" http://localhost:8080/default/e2e-v1beta1-knative)
  if test "$res" = "200"; then
    echo "knative: success" | ./bin/yq
    kubectl delete -f ${MANIFESTS_PATH} > /dev/null 2>&1
  fi
fi

pkill -9 -f "kubectl port-forward"
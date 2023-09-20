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

echo "---Status of all related pods---"
kubectl get po -n projectcontour
kubectl get po -n dapr-system
kubectl get po -n keda
kubectl get po -n knative-serving
kubectl get po -n kourier-system
kubectl get po -n openfunction
kubectl get po -n shipwright-build
kubectl get po -n tekton-pipelines

echo "---Gateways---"
kubectl get gateways.networking.openfunction.io -n openfunction openfunction -oyaml

echo "---Functions---"
kubectl get fn -oyaml

echo "---Builder---"
kubectl get builder -oyaml
kubectl get build -oyaml
kubectl get buildrun -oyaml

echo "---Serving---"
kubectl get serving -oyaml

echo "---OpenFunction controller pod logs---"
kubectl logs -n openfunction "$(kubectl get pod -n openfunction -o jsonpath='{.items[0].metadata.name}')" openfunction

echo "---OpenFunction controller pod status---"
kubectl describe po -n openfunction "$(kubectl get pod -n openfunction -o jsonpath='{.items[0].metadata.name}')"

echo "---Shipwright controller pod logs---"
kubectl logs -n shipwright-build "$(kubectl get pod -n shipwright-build -o jsonpath='{.items[0].metadata.name}')"

echo "---Knative controller pod logs---"
kubectl logs -n knative-serving "$(kubectl get pod -n knative-serving -l app=controller -o jsonpath='{.items[0].metadata.name}')" controller


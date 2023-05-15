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

#!/bin/bash -e

TEMP_DIR="cert_temp"

mkdir -p ${TEMP_DIR}

CN="openfunction-webhook-service"
SAN_IP=""
SAN_DNS="openfunction-webhook-service,openfunction-webhook-service.openfunction,openfunction-webhook-service.openfunction.svc,openfunction-webhook-service.openfunction.svc.cluster.local"
C=CN
SSL_SIZE=2048
EXPIRE=${EXPIRE:-3650}
SSL_CONFIG=${TEMP_DIR}/'openssl.cnf'
CA_KEY=${TEMP_DIR}/${CA_KEY-"cakey.pem"}
CA_CERT=${TEMP_DIR}/${CA_CERT-"cacerts.pem"}
CA_SUBJECT=ca-$CN
SSL_KEY=${TEMP_DIR}/$CN.key
SSL_CSR=${TEMP_DIR}/$CN.csr
SSL_CERT=${TEMP_DIR}/$CN.crt
SSL_SUBJECT=${CN}

export K8S_SECRET_COMBINE_CA=${K8S_SECRET_COMBINE_CA:-'true'}

# Generate CA Key
openssl genrsa -out ${CA_KEY} ${SSL_SIZE} > /dev/null

# Generate CA Certificate
openssl req -x509 -sha256 -new -nodes -key ${CA_KEY} \
    -days ${EXPIRE} -out ${CA_CERT} -subj "/CN=${CA_SUBJECT}" > /dev/null || exit 1

# Generate SSL config
cat > ${SSL_CONFIG} <<EOM
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth, serverAuth
EOM

if [[ -n ${SAN_DNS} || -n ${SAN_IP} ]]; then
    cat >> ${SSL_CONFIG} <<EOM
subjectAltName = @alt_names
[alt_names]
EOM
    IFS=","
    dns=(${SAN_DNS})
    dns+=(${SSL_SUBJECT})
    for i in "${!dns[@]}"; do
      echo DNS.$((i+1)) = ${dns[$i]} >> ${SSL_CONFIG}
    done

    if [[ -n ${SAN_IP} ]]; then
        ip=(${SAN_IP})
        for i in "${!ip[@]}"; do
          echo IP.$((i+1)) = ${ip[$i]} >> ${SSL_CONFIG}
        done
    fi
fi

# Generate SSL key
openssl genrsa -out ${SSL_KEY} ${SSL_SIZE} > /dev/null || exit 1

# Generate SSL csr
openssl req -sha256 -new -key ${SSL_KEY} -out ${SSL_CSR} \
-subj "/CN=${SSL_SUBJECT}" -config ${SSL_CONFIG} > /dev/null || exit 1

# Generate SSL cert
openssl x509 -sha256 -req -in ${SSL_CSR} -CA ${CA_CERT} \
    -CAkey ${CA_KEY} -CAcreateserial -out ${SSL_CERT} \
    -days ${EXPIRE} -extensions v3_req \
    -extfile ${SSL_CONFIG} > /dev/null || exit 1

# Update caBundle and tls.*
sed -ri "s/(tls.crt: )[^\n]*/\1$(cat ${SSL_CERT} | base64 -w 0)/" ../config/cert/webhook-server-cert.yaml
sed -ri "s/(tls.key: )[^\n]*/\1$(cat ${SSL_KEY} | base64 -w 0)/" ../config/cert/webhook-server-cert.yaml
sed -ri "s/(caBundle: )[^\n]*/\1$(cat ${CA_CERT} | base64 -w 0)/" ../config/webhook/manifests.yaml
sed -ri "s/(caBundle: )[^\n]*/\1$(cat ${CA_CERT} | base64 -w 0)/" ../config/crd/patches/webhook_in_builders.yaml
sed -ri "s/(caBundle: )[^\n]*/\1$(cat ${CA_CERT} | base64 -w 0)/" ../config/crd/patches/webhook_in_functions.yaml
sed -ri "s/(caBundle: )[^\n]*/\1$(cat ${CA_CERT} | base64 -w 0)/" ../config/crd/patches/webhook_in_servings.yaml

# Clear temp
rm -rf ${TEMP_DIR}
#!/bin/bash

set -e

kind_required_version=0.5.0
kind_node_image="node:v1.13.10@sha256:2f5f882a6d0527a2284d29042f3a6a07402e1699d792d0d5a9b9a48ef155fa2a"

## Test Infra Setup
## This will use Kind, Reckoner, and Helm to setup a test infrastructure locally for astro

function version_gt() {
    test "$(printf '%s\n' "$@" | sort -V | head -n 1)" != "$1";
}

cd "$( cd "$(dirname "$0")" ; pwd -P )"

required_clis="reckoner helm kind"
for cli in $required_clis; do
  command -v "$cli" >/dev/null 2>&1 || { echo >&2 "I require $cli but it's not installed.  Aborting."; exit 1; }
done

kind_version=$(kind version | cut -d+ -f1)

if version_gt "$kind_required_version" "$kind_version"; then
     echo "This script requires kind version greater than or equal to $kind_required_version!"
     exit 1
fi

## Create the kind cluster
kind create cluster \
  --config kind.yaml \
  --name test-infra \
  --image="kindest/$kind_node_image" || true

# shellcheck disable=SC2034
KUBECONFIG="$(kind get kubeconfig-path --name="test-infra")"
until kubectl cluster-info; do
    echo "Waiting for cluster to become available...."
    sleep 3
done

## Helm Init
kubectl -n kube-system create sa tiller --dry-run -o yaml --save-config | kubectl apply -f -;
kubectl create clusterrolebinding tiller --clusterrole cluster-admin --serviceaccount="kube-system:tiller" --serviceaccount=kube-system:tiller -o yaml --dry-run | kubectl -n "kube-system" apply -f -

helm init --wait --upgrade --service-account tiller

## Reckoner
if [ -z ${CIRCLE_SHA1} ]
then
  CIRCLE_SHA1='v1.2.0' reckoner plot course.yml
else
  reckoner plot course.yml
fi

echo "Use 'kind get kubeconfig-path --name=test-infra' to get your kubeconfig"
echo "When done, use 'kind delete cluster --name test-infra to remove the cluster"

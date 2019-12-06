#!/bin/bash

printf "\n\n"
echo "**************************"
echo "** Begin E2E Test Setup **"
echo "**************************"
printf "\n\n"

set -e

printf "\n\n"
echo "********************************************************************"
echo "** Install Astro at $CI_SHA1 **"
echo "********************************************************************"
printf "\n\n"

kubectl create ns astro
kubectl -n astro apply -f /hack/manifests/

kubectl -n astro wait deployment --timeout=60s --for condition=available -l app.kubernetes.io/name=astro

kubectl get pods --all-namespaces

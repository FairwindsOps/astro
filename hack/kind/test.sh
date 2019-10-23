#!/bin/bash

set -e


printf "\n\n"
echo "**********************************"
echo "** Wait for deploy availability **"
echo "**********************************"
printf "\n\n"
kubectl -n astro wait deployment --timeout=60s --for condition=available -l app.kubernetes.io/name=astro

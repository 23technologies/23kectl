#! /usr/bin/env bash

if [ ! -v KUBECONFIG ]; then exit 1; fi

read -sp "Please enter the password to use for the admin user" adminPassword

podCidr=$(kubectl get node -o jsonpath='{ .items[0].spec.podCIDR }')
serviceCidr=$(echo '{"apiVersion":"v1","kind":"Service","metadata":{"name":"tst"},"spec":{"clusterIP":"1.1.1.1","ports":[{"port":443}]}}' | kubectl apply -f - 2>&1 | sed 's/.*valid IPs is //')
clusterIp=$(nmap -sL -n $serviceCidr | awk '/Nmap scan report/{print $NF}' | sed '100q;d')


helm template 23ke-env-configuration . -f values.yaml --set clusterIP=$clusterIp,seed.networks.services=$serviceCidr,seed.networks.pods=$podCidr,adminPassword=$adminPassword --output-dir . > /dev/null

mv 23ke-env-configuration/templates/* 23ke-env-configuration
rm -r 23ke-env-configuration/templates/

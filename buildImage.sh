#!/usr/bin/env bash
docker build -t www.registry.it/podinteractive .
docker push www.registry.it/podinteractive
kubectl apply -f deploy/deploy.yaml

kubectl scale deploy podinteractive -n kube-system --replicas=0
kubectl scale deploy podinteractive -n kube-system --replicas=1
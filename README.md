# k8s resource collector

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

K8s resource collector subscribes to the changes in the K8s API server for a specified set of resource types. By default, it collects updates to the following resource types.

- configMap
- cronJob
- daemonSet
- deployment
- event
- job
- namespace
- node
- persistentVolumeClaim
- persistentVolume
- pod
- podTemplate
- replicaSet
- resourceQuota
- secret
- serviceAccount
- service
- statefulSet
- ingressClass
- ingress
- networkPolicy

For configmaps and secrets, the data field is deleted since it may contain sensitive information.

## Deploy to your cluster

```bash
kubectl apply -f manifests/k8s-resource-collector.yaml
```

## Uninstall

```bash
kubectl delete -f manifests/k8s-resource-collector.yaml
```

## Stream to webb.ai

You will need to edit the `CLIENT_ID` and `API_KEY` env var in manifests/k8s-resource-collector.yaml to stream the data to webb.ai.
Reach out to us to get a CLIENT_ID and API_KEY.

## See staged data
```bash
pod_name=$(kubectl get pods -n webbai | grep resource-collector | awk '{print $1}')
kubectl exec --stdin --tty $pod_name -n webbai -- /bin/sh
cd /app/data/
cat k8s_resource.log
```

Each row of `k8s_resource.log` is a json. 

# k8s resource collector

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

The k8s resource collector is a tool that subscribes to changes in the Kubernetes API server for a specified set of resource types. It provides an easy way to collect and store the data related to these resources.

## Supported Resource Types

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

## Deploy to your cluster

```bash
kubectl manifests/k8s-resource-collector.yaml
```

## See staged data
```bash
kubectl exec --stdin --tty <resource-collector-pod> -n webbai -- /bin/sh
cd /app/data/
```

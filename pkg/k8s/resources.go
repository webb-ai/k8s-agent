package k8s

import "k8s.io/apimachinery/pkg/runtime/schema"

var WatchedGVRs = []schema.GroupVersionResource{
	// Group, Version, Resource

	{Group: "", Version: "v1", Resource: "events"},
	{Group: "", Version: "v1", Resource: "namespaces"},
	{Group: "", Version: "v1", Resource: "nodes"},
	{Group: "", Version: "v1", Resource: "persistentvolumeclaims"},
	{Group: "", Version: "v1", Resource: "persistentvolumes"},
	{Group: "", Version: "v1", Resource: "pods"},
	{Group: "", Version: "v1", Resource: "resourcequotas"},
	{Group: "", Version: "v1", Resource: "serviceaccounts"},
	{Group: "", Version: "v1", Resource: "services"},

	{Group: "apps", Version: "v1", Resource: "daemonsets"},
	{Group: "apps", Version: "v1", Resource: "deployments"},
	{Group: "apps", Version: "v1", Resource: "replicasets"},
	{Group: "apps", Version: "v1", Resource: "statefulsets"},

	{Group: "batch", Version: "v1", Resource: "cronjobs"},
	{Group: "batch", Version: "v1", Resource: "jobs"},

	{Group: "networking.k8s.io", Version: "v1", Resource: "ingressclasses"},
	{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
	{Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"},
}

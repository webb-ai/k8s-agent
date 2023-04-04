package k8s

import "k8s.io/apimachinery/pkg/runtime/schema"

var configMapGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
var cronjobGVR = schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"}
var daemonsetGVR = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}
var deploymentGVR = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
var eventGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "events"}
var jobGVR = schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"}
var namespaceGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}
var nodeGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}
var persistentvolumeclaimGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumeclaims"}
var persistentvolumeGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumes"}
var podGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
var podtemplateGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "podtemplates"}
var replicasetGVR = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"}
var resourcequotaGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "resourcequotas"}
var secretGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}
var serviceaccountGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"}
var serviceGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
var statefulsetGVR = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}
var ingressclassGVR = schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingressclasses"}
var ingressGVR = schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"}
var networkpolicyGVR = schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"}

var WatchedGVRs = []schema.GroupVersionResource{
	// configMapGVR,
	cronjobGVR,
	daemonsetGVR,
	deploymentGVR,
	eventGVR,
	jobGVR,
	namespaceGVR,
	nodeGVR,
	persistentvolumeclaimGVR,
	persistentvolumeGVR,
	podGVR,
	podtemplateGVR,
	replicasetGVR,
	resourcequotaGVR,
	// secretGVR,
	serviceaccountGVR,
	serviceGVR,
	statefulsetGVR,
	ingressclassGVR,
	ingressGVR,
	networkpolicyGVR,
}

var WorkloadGVRs = []schema.GroupVersionResource{
	podGVR,
	nodeGVR,
	daemonsetGVR,
	deploymentGVR,
	statefulsetGVR,
	jobGVR,
	cronjobGVR,
}

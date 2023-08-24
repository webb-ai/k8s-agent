package k8s

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/klog/v2"
)

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
var hpaGVR = schema.GroupVersionResource{Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"}
var vpaGVR = schema.GroupVersionResource{Group: "autoscaling.k8s.io", Version: "v1", Resource: "verticalpodautoscalers"}
var kedaScaledObjectGVR = schema.GroupVersionResource{Group: "keda.sh", Version: "v1alpha1", Resource: "ScaledObject"}
var mutatingWebhookGVR = schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "mutatingwebhookconfigurations"}
var validatingWebhookGVR = schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "validatingwebhookconfigurations"}

var WatchedGVRs = []schema.GroupVersionResource{
	configMapGVR,
	cronjobGVR,
	daemonsetGVR,
	deploymentGVR,
	jobGVR,
	namespaceGVR,
	nodeGVR,
	persistentvolumeclaimGVR,
	persistentvolumeGVR,
	podGVR,
	podtemplateGVR,
	replicasetGVR,
	resourcequotaGVR,
	secretGVR,
	serviceaccountGVR,
	serviceGVR,
	statefulsetGVR,
	ingressclassGVR,
	ingressGVR,
	networkpolicyGVR,
	hpaGVR,
	vpaGVR,
	kedaScaledObjectGVR,
	mutatingWebhookGVR,
	validatingWebhookGVR,
}

var BackupGVRs = []schema.GroupVersionResource{
	podGVR,
	serviceGVR,
	deploymentGVR,
	statefulsetGVR,
	daemonsetGVR,
	jobGVR,
	cronjobGVR,
	namespaceGVR,
}

func GetAllResources(discoveryClient discovery.ServerResourcesInterface) (map[schema.GroupVersionResource]struct{}, error) {
	resources, err := discoveryClient.ServerPreferredResources()
	if err != nil {
		return nil, err
	}

	result := make(map[schema.GroupVersionResource]struct{})

	for _, resourcesList := range resources {
		gv, err := schema.ParseGroupVersion(resourcesList.GroupVersion)
		if err != nil {
			klog.Warningf("%w", err)
		}
		for _, resource := range resourcesList.APIResources {
			gvr := schema.GroupVersionResource{Group: gv.Group, Version: gv.Version, Resource: resource.Name}
			result[gvr] = struct{}{}
		}
	}

	return result, nil
}

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
var endpointSliceGVR = schema.GroupVersionResource{Group: "discovery.k8s.io", Version: "v1", Resource: "endpointslices"}
var endpointsGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "endpoints"}
var eventGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "events"}
var hpaGVR = schema.GroupVersionResource{Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"}
var ingressGVR = schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"}
var ingressclassGVR = schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingressclasses"}
var jobGVR = schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"}
var kedaScaledObjectGVR = schema.GroupVersionResource{Group: "keda.sh", Version: "v1alpha1", Resource: "scaledobjects"}
var mutatingWebhookGVR = schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "mutatingwebhookconfigurations"}
var namespaceGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}
var networkpolicyGVR = schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"}
var nodeGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}
var persistentvolumeGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumes"}
var persistentvolumeclaimGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumeclaims"}
var podGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
var podtemplateGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "podtemplates"}
var priorityclassGVR = schema.GroupVersionResource{Group: "scheduling.k8s.io", Version: "v1", Resource: "priorityclasses"}
var replicasetGVR = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"}
var resourcequotaGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "resourcequotas"}
var secretGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}
var serviceGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
var serviceaccountGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"}
var statefulsetGVR = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}
var validatingWebhookGVR = schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "validatingwebhookconfigurations"}
var vpaGVR = schema.GroupVersionResource{Group: "autoscaling.k8s.io", Version: "v1", Resource: "verticalpodautoscalers"}

var WatchedGVRs = []schema.GroupVersionResource{
	configMapGVR,
	cronjobGVR,
	daemonsetGVR,
	deploymentGVR,
	endpointSliceGVR,
	endpointsGVR,
	hpaGVR,
	ingressGVR,
	ingressclassGVR,
	jobGVR,
	kedaScaledObjectGVR,
	mutatingWebhookGVR,
	namespaceGVR,
	networkpolicyGVR,
	nodeGVR,
	persistentvolumeGVR,
	persistentvolumeclaimGVR,
	podGVR,
	podtemplateGVR,
	priorityclassGVR,
	replicasetGVR,
	resourcequotaGVR,
	secretGVR,
	serviceGVR,
	serviceaccountGVR,
	statefulsetGVR,
	validatingWebhookGVR,
	vpaGVR,
}

var BackupGVRs = []schema.GroupVersionResource{
	cronjobGVR,
	daemonsetGVR,
	deploymentGVR,
	endpointSliceGVR,
	endpointsGVR,
	jobGVR,
	namespaceGVR,
	nodeGVR,
	podGVR,
	serviceGVR,
	statefulsetGVR,
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

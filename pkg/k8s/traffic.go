package k8s

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/rs/zerolog"
	"github.com/webb-ai/k8s-agent/pkg/api"
	"github.com/webb-ai/k8s-agent/pkg/traffic"
	"github.com/webb-ai/k8s-agent/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/klog/v2"
)

type TrafficCollector struct {
	informerFactory dynamicinformer.DynamicSharedInformerFactory
	interval        time.Duration
	podSelector     labels.Selector
	serverPort      int
	metricsPort     int
	logger          zerolog.Logger
	client          api.Client
}

func NewTrafficCollector(
	informerFactory dynamicinformer.DynamicSharedInformerFactory,
	interval time.Duration,
	podSelector labels.Selector,
	serverPort,
	metricsPort int,
	logger zerolog.Logger,
	client api.Client,
) *TrafficCollector {
	return &TrafficCollector{
		informerFactory: informerFactory,
		interval:        interval,
		podSelector:     podSelector,
		serverPort:      serverPort,
		metricsPort:     metricsPort,
		logger:          logger,
		client:          client,
	}
}

func (c *TrafficCollector) Start(ctx context.Context) error {
	if c.podSelector == nil {
		klog.Warningf("no traffic pod selector specified, traffic metrics collection won't run")
		return nil
	}

	klog.Infof("starting to collect traffic metrics every %v for pods with labels %v", c.interval, c.podSelector)
	c.setTargetPods()
	for {
		select {
		case <-time.After(c.interval):
			c.setServiceIps()
			c.setTargetPods()
			c.collectMetrics()
		case <-ctx.Done():
			return nil
		}
	}
}

func (c *TrafficCollector) collectMetrics() {
	pods, err := c.informerFactory.ForResource(podGVR).Lister().List(c.podSelector)

	if err != nil {
		klog.Error(err)
		return
	}

	if len(pods) == 0 {
		klog.Error("no traffic collector found, skipping ...")
		return
	}

	for _, podRuntimeObject := range pods {
		pod, err := util.UnstructuredToPod(podRuntimeObject.(*unstructured.Unstructured))
		if err != nil {
			klog.Error(err)
			continue
		}
		podIp := pod.Status.PodIP
		metricsUrl := fmt.Sprintf("http://%s:%d/webbai_metrics", podIp, c.metricsPort)
		klog.Infof("scraping %s for prometheus metrics", metricsUrl)
		metricText, metricFamilies, err := traffic.ScrapeTarget(metricsUrl)
		if err != nil {
			klog.Error(err)
		}

		c.logger.Info().Any("payload", metricText).Msg("metrics")
		writeRequest := traffic.MetricFamiliesToProtoWriteRequest(metricFamilies)
		err = c.client.SendTrafficMetrics(writeRequest)
		if err != nil {
			klog.Error(err)
		}
	}
}

func (c *TrafficCollector) setTargetPods() {
	var allRunningPods []*corev1.Pod
	var trafficCollectorPods []*corev1.Pod
	pods, err := c.informerFactory.ForResource(podGVR).Lister().List(labels.Everything())
	if err != nil {
		klog.Error(err)
		return
	}

	for _, podRuntimeObject := range pods {
		pod, _ := util.UnstructuredToPod(podRuntimeObject.(*unstructured.Unstructured))
		if pod.Status.Phase == corev1.PodRunning && !c.podSelector.Matches(labels.Set(pod.Labels)) {
			klog.Infof("targeting pod %s from namespace %s", pod.Name, pod.Namespace)
			pod.ManagedFields = nil
			allRunningPods = append(allRunningPods, pod)
		}
		if pod.Status.Phase == corev1.PodRunning && c.podSelector.Matches(labels.Set(pod.Labels)) {
			trafficCollectorPods = append(trafficCollectorPods, pod)
		}
	}

	for _, pod := range trafficCollectorPods {
		podIp := pod.Status.PodIP
		podTargetsUrl := fmt.Sprintf("http://%s:%d/pods/set-targeted", podIp, c.serverPort)
		klog.Infof("setting pod targets to %s", podTargetsUrl)
		err = traffic.SetPodTargets(allRunningPods, podTargetsUrl)
		if err != nil {
			klog.Error(err)
		}
	}
}

func (c *TrafficCollector) setServiceIps() {
	serviceByIp := make(map[string]string)
	serviceByClusterIp := make(map[string]string)
	c.collectServiceIps(serviceByIp)
	c.collectClusterIps(serviceByClusterIp)

	pods, err := c.informerFactory.ForResource(podGVR).Lister().List(c.podSelector)
	if err != nil {
		klog.Error(err)
		return
	}

	for _, podRuntimeObject := range pods {
		pod, _ := util.UnstructuredToPod(podRuntimeObject.(*unstructured.Unstructured))

		if pod.Status.Phase == corev1.PodRunning {
			podIp := pod.Status.PodIP
			targetUrl := fmt.Sprintf("http://%s:%d/service_ips", podIp, c.serverPort)
			klog.Infof("setting pod ips to %s", targetUrl)
			err = traffic.SetServiceIps(serviceByIp, serviceByClusterIp, targetUrl)
			if err != nil {
				klog.Error(err)
			}
		}
	}

	klog.Infof("new ip mapping: %v", serviceByIp)
}

func (c *TrafficCollector) collectClusterIps(serviceByClusterIp map[string]string) {
	serviceObjects, err := c.informerFactory.ForResource(serviceGVR).Lister().List(labels.Everything())
	if err != nil {
		klog.Errorf("error fetching services: %w", err)
	}

	for _, serviceObject := range serviceObjects {
		unstr := serviceObject.(*unstructured.Unstructured)
		service, _ := util.UnstructuredToService(unstr)
		selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: service.Spec.Selector})
		if err != nil {
			klog.Warningf("unexpected error: cannot convert %w", err)
			continue
		}

		clusterIp := util.GetClusterIP(unstr)
		objectId := c.getObjectIdBySelector(service.Namespace, selector)

		if objectId != "" && clusterIp != "" {
			serviceByClusterIp[clusterIp] = objectId
		}
	}
}

func (c *TrafficCollector) collectServiceIps(serviceByIp map[string]string) {
	gvrs := []schema.GroupVersionResource{deploymentGVR, statefulsetGVR, daemonsetGVR, jobGVR}

	for _, gvr := range gvrs {
		runtimeObjects, err := c.informerFactory.ForResource(gvr).Lister().List(labels.Everything())
		if err != nil {
			klog.Errorf("error fetching %s: %w", gvr, err)
			continue
		}
		for _, runtimeObject := range runtimeObjects {
			unstr := runtimeObject.(*unstructured.Unstructured)
			labelSelector, err := util.GetLabelSelector(unstr)
			if err != nil {
				klog.Errorf("cannot get label selector: %w", err)
				continue
			}
			selector, _ := metav1.LabelSelectorAsSelector(labelSelector)
			objectId := c.getObjectId(unstr, gvr)
			c.updatePodIps(selector, objectId, serviceByIp, unstr.GetNamespace())
		}
	}
}

func (c *TrafficCollector) getObjectId(
	unstr *unstructured.Unstructured,
	gvr schema.GroupVersionResource,
) string {
	var objectId = generateId(unstr)
	if gvr == jobGVR && len(unstr.GetOwnerReferences()) != 0 {
		ownerRef := unstr.GetOwnerReferences()[0]
		if ownerRef.Kind == "CronJob" && ownerRef.APIVersion == "batch/v1" {
			objectId = fmt.Sprintf("batch/v1|CronJob|%s|%s", unstr.GetNamespace(), ownerRef.Name)
		}
	}
	return objectId
}

func (c *TrafficCollector) getObjectIdBySelector(
	namespace string,
	selector labels.Selector,
) string {
	gvrs := []schema.GroupVersionResource{deploymentGVR, statefulsetGVR, daemonsetGVR}
	for _, gvr := range gvrs {
		objects, err := c.informerFactory.ForResource(gvr).Lister().ByNamespace(namespace).List(selector)
		if err != nil {
			klog.Errorf("error fetching %s: %w", gvr, err)
			continue
		}
		if len(objects) == 1 {
			object := objects[0]
			return generateId(object.(*unstructured.Unstructured))
		}
	}
	return ""
}

func (c *TrafficCollector) updatePodIps(
	selector labels.Selector, objectId string, mapping map[string]string, namespace string) {
	pods, err := c.informerFactory.ForResource(podGVR).Lister().ByNamespace(namespace).List(selector)

	if err != nil {
		klog.Warningf("error fetching pods: %w", err)
		return
	}

	for _, podObject := range pods {
		pod, _ := util.UnstructuredToPod(podObject.(*unstructured.Unstructured))
		// only consider pods whose ip is different from host ip
		if pod.Status.PodIP != pod.Status.HostIP {
			mapping[pod.Status.PodIP] = objectId
		}
	}
}

func generateId(unstr *unstructured.Unstructured) string {
	return fmt.Sprintf("%s|%s|%s|%s", unstr.GetAPIVersion(), unstr.GetKind(), unstr.GetNamespace(), unstr.GetName())
}

package k8s

import (
	"context"
	"fmt"
	"time"

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

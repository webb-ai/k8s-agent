package k8s

import (
	"context"
	"time"

	"github.com/webb-ai/k8s-agent/pkg/util"

	"github.com/webb-ai/k8s-agent/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rs/zerolog"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

type Collector struct {
	defaultResyncPeriod        time.Duration
	resourceCollectionInterval time.Duration
	dynamicClient              dynamic.Interface
	logger                     zerolog.Logger
	client                     api.Client
	informerFactory            dynamicinformer.DynamicSharedInformerFactory
	metrics                    *Metrics
}

func NewCollector(
	defaultResyncPeriod time.Duration,
	resourceCollectionPeriod time.Duration,
	dynamicClient dynamic.Interface,
	logger zerolog.Logger,
	client api.Client,
) *Collector {
	return &Collector{
		defaultResyncPeriod:        defaultResyncPeriod,
		resourceCollectionInterval: resourceCollectionPeriod,
		dynamicClient:              dynamicClient,
		logger:                     logger,
		client:                     client,
		metrics:                    NewMetrics(),
	}
}

func (c *Collector) OnAdd(obj interface{}) {
	// TODO: retry on retryable errors
	runtimeObject, err := util.InterfacetoUnstructured(obj)
	if err != nil {
		klog.Error(err)
		return
	}

	event := api.NewResourceChangeEvent(nil, runtimeObject)
	c.logger.Info().Any("payload", event).Msg("object_add")

	_ = c.client.SendK8sChangeEvent(event)

	c.metrics.ChangeEventCounter.With(
		map[string]string{
			EventTypeKey:  "object_add",
			ObjectKindKey: runtimeObject.GetKind(),
		},
	).Inc()
}

func (c *Collector) OnDelete(obj interface{}) {
	runtimeObject, err := util.InterfacetoUnstructured(obj)
	if err != nil {
		klog.Error(err)
		return
	}

	event := api.NewResourceChangeEvent(runtimeObject, nil)

	c.logger.Info().Any("payload", event).Msg("object_delete")
	_ = c.client.SendK8sChangeEvent(event)
	c.metrics.ChangeEventCounter.With(
		map[string]string{
			EventTypeKey:  "object_delete",
			ObjectKindKey: runtimeObject.GetKind(),
		},
	).Inc()
}

func (c *Collector) OnUpdate(oldObj, newObj interface{}) {
	oldObject, err := util.InterfacetoUnstructured(oldObj)
	if err != nil {
		klog.Error(err)
		return
	}

	newObject, err := util.InterfacetoUnstructured(newObj)
	if err != nil {
		klog.Error(err)
		return
	}

	if util.IsConfigMapOrSecret(oldObject) && !util.HasDataChanged(oldObject, newObject) {
		// if a configmap or secret, and the data hasn't changed, skip
		return
	}

	if oldObject.GetResourceVersion() != newObject.GetResourceVersion() || util.HasStatusChanged(oldObject, newObject) {
		klog.Infof("detected resource version change or status change of %s/%s(%s)",
			newObject.GetNamespace(), newObject.GetName(), newObject.GroupVersionKind())
		event := api.NewResourceChangeEvent(oldObject, newObject)
		c.logger.Info().Any("payload", event).Msg("object_update")

		_ = c.client.SendK8sChangeEvent(event)

		c.metrics.ChangeEventCounter.With(
			map[string]string{
				EventTypeKey:  "object_update",
				ObjectKindKey: oldObject.GetKind(),
			},
		).Inc()
	}

}

func (c *Collector) Start(ctx context.Context) error {
	klog.Infof("starting k8s resource collector process")
	c.startWorkloadCollectionLoop(ctx)

	c.informerFactory = dynamicinformer.NewDynamicSharedInformerFactory(c.dynamicClient, c.defaultResyncPeriod)

	eventHandler := cache.ResourceEventHandlerFuncs{
		AddFunc:    c.OnAdd,
		UpdateFunc: c.OnUpdate,
		DeleteFunc: c.OnDelete,
	}

	for _, gvr := range WatchedGVRs {
		klog.Infof("starting to watch for resource %v", gvr)
		informer := c.informerFactory.ForResource(gvr)
		_, err := informer.Informer().AddEventHandler(eventHandler)
		if err != nil {
			klog.Warningf("unable to watch for resource %v: %v", gvr, err)
		}
	}
	c.informerFactory.WaitForCacheSync(ctx.Done())
	c.informerFactory.Start(ctx.Done())

	<-ctx.Done()
	klog.Infof("stopped k8s resource collector process")
	return nil
}

func (c *Collector) startWorkloadCollectionLoop(ctx context.Context) {
	klog.Infof("starting to collect workload resources every %v", c.resourceCollectionInterval)

	go func() {
		c.collectWorkloadResourcesAndEvents(ctx)
		for {
			select {
			case <-time.After(c.resourceCollectionInterval):
				c.collectWorkloadResourcesAndEvents(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (c *Collector) collectWorkloadResourcesAndEvents(ctx context.Context) {
	for _, gvr := range WorkloadAndEventGVRs {
		klog.Infof("listing all resources for %v", gvr)
		listResult, err := c.dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
		if err != nil {
			klog.Error(err)
		} else if len(listResult.Items) > 0 {
			c.logger.Info().Any("payload", listResult.Items).Msg("resource_list")
			_ = c.client.SendK8sResources(api.NewResourceList(listResult.Items))
		}
	}
}

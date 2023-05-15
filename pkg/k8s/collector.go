package k8s

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/webb-ai/k8s-agent/pkg/util"

	"github.com/rs/zerolog"
	"github.com/webb-ai/k8s-agent/pkg/api"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

type Collector struct {
	defaultResyncPeriod     time.Duration
	eventCollectionInterval time.Duration
	dynamicClient           dynamic.Interface
	logger                  zerolog.Logger
	client                  api.Client
	informerFactory         dynamicinformer.DynamicSharedInformerFactory
	metrics                 *Metrics
}

func NewCollector(
	defaultResyncPeriod time.Duration,
	eventCollectionInterval time.Duration,
	dynamicClient dynamic.Interface,
	logger zerolog.Logger,
	client api.Client,
) *Collector {
	return &Collector{
		defaultResyncPeriod:     defaultResyncPeriod,
		eventCollectionInterval: eventCollectionInterval,
		dynamicClient:           dynamicClient,
		logger:                  logger,
		client:                  client,
		metrics:                 NewMetrics(),
	}
}

func (c *Collector) noOp(obj interface{}) {

}

func (c *Collector) noOpUpdate(oldObj, newObj interface{}) {

}

func (c *Collector) OnAdd(obj interface{}) {
	// TODO: retry on retryable errors
	runtimeObject, err := util.InterfaceToUnstructured(obj)
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
	runtimeObject, err := util.InterfaceToUnstructured(obj)
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
	oldObject, err := util.InterfaceToUnstructured(oldObj)
	if err != nil {
		klog.Error(err)
		return
	}

	newObject, err := util.InterfaceToUnstructured(newObj)
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

func (c *Collector) addHandlerForGvr(gvr schema.GroupVersionResource, handler cache.ResourceEventHandler) {
	klog.Infof("starting to watch for resource %v", gvr)
	informer := c.informerFactory.ForResource(gvr)
	_, err := informer.Informer().AddEventHandler(handler)
	if err != nil {
		klog.Warningf("unable to watch for resource %v: %v", gvr, err)
	}
}

func (c *Collector) Start(ctx context.Context) error {
	klog.Infof("starting k8s resource collector process")
	c.informerFactory = dynamicinformer.NewDynamicSharedInformerFactory(c.dynamicClient, c.defaultResyncPeriod)

	handler := cache.ResourceEventHandlerFuncs{
		AddFunc:    c.OnAdd,
		UpdateFunc: c.OnUpdate,
		DeleteFunc: c.OnDelete,
	}

	for _, gvr := range WatchedGVRs {
		c.addHandlerForGvr(gvr, handler)
	}

	noOpHandler := cache.ResourceEventHandlerFuncs{
		AddFunc:    c.noOp,
		UpdateFunc: c.noOpUpdate,
		DeleteFunc: c.noOp,
	}

	c.addHandlerForGvr(eventGVR, noOpHandler) // only keep events in the cache, do not handle

	c.informerFactory.WaitForCacheSync(ctx.Done())
	c.informerFactory.Start(ctx.Done())
	c.startWorkloadCollectionLoop(ctx)

	<-ctx.Done()
	klog.Infof("stopped k8s resource collector process")
	return nil
}

func (c *Collector) startWorkloadCollectionLoop(ctx context.Context) {
	klog.Infof("starting to collect workload resources every %v", c.eventCollectionInterval)

	go func() {
		for {
			select {
			case <-time.After(c.eventCollectionInterval):
				c.collectWorkloadResourcesAndEvents()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (c *Collector) collectWorkloadResourcesAndEvents() {
	for _, gvr := range WorkloadAndEventGVRs {
		klog.Infof("listing all resources for %v", gvr)
		listResult, err := c.informerFactory.ForResource(gvr).Lister().List(labels.Everything())
		if err != nil {
			klog.Error(err)
		} else if len(listResult) > 0 {
			c.logger.Info().Any("payload", listResult).Msg("resource_list")
			_ = c.client.SendK8sResources(api.NewResourceList(listResult))
		} else {
			klog.Infof("no result for %v", gvr)
		}
	}
}

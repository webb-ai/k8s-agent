package k8s

import (
	"context"
	"time"

	"k8s.io/client-go/discovery"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/webb-ai/k8s-agent/pkg/util"

	"github.com/rs/zerolog"
	"github.com/webb-ai/k8s-agent/pkg/api"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

type ChangeCollector struct {
	eventCollectionInterval  time.Duration
	backupCollectionInterval time.Duration
	informerFactory          dynamicinformer.DynamicSharedInformerFactory
	discoveryClient          discovery.ServerResourcesInterface
	logger                   zerolog.Logger
	client                   api.Client
	metrics                  *Metrics
}

func NewChangeCollector(
	eventCollectionInterval time.Duration,
	backupCollectionInterval time.Duration,
	informerFactory dynamicinformer.DynamicSharedInformerFactory,
	discoveryClient discovery.ServerResourcesInterface,
	logger zerolog.Logger,
	client api.Client,
) *ChangeCollector {
	return &ChangeCollector{
		eventCollectionInterval:  eventCollectionInterval,
		backupCollectionInterval: backupCollectionInterval,
		informerFactory:          informerFactory,
		discoveryClient:          discoveryClient,
		logger:                   logger,
		client:                   client,
		metrics:                  NewMetrics(),
	}
}

func (c *ChangeCollector) noOp(obj interface{}) {

}

func (c *ChangeCollector) noOpUpdate(oldObj, newObj interface{}) {

}

func (c *ChangeCollector) OnAdd(obj interface{}) {
	// TODO: retry on retryable errors
	runtimeObject, err := util.InterfaceToUnstructured(obj)
	if err != nil {
		klog.Error(err)
		return
	}

	event := api.NewK8sChangeEvent(nil, runtimeObject)
	c.logger.Info().Any("payload", event).Msg("object_add")

	_ = c.client.SendChangeEvent(event)

	c.metrics.ChangeEventCounter.With(
		map[string]string{
			EventTypeKey:  "object_add",
			ObjectKindKey: runtimeObject.GetKind(),
		},
	).Inc()
}

func (c *ChangeCollector) OnDelete(obj interface{}) {
	runtimeObject, err := util.InterfaceToUnstructured(obj)
	if err != nil {
		klog.Error(err)
		return
	}

	event := api.NewK8sChangeEvent(runtimeObject, nil)

	c.logger.Info().Any("payload", event).Msg("object_delete")
	_ = c.client.SendChangeEvent(event)
	c.metrics.ChangeEventCounter.With(
		map[string]string{
			EventTypeKey:  "object_delete",
			ObjectKindKey: runtimeObject.GetKind(),
		},
	).Inc()
}

func (c *ChangeCollector) OnUpdate(oldObj, newObj interface{}) {
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
		event := api.NewK8sChangeEvent(oldObject, newObject)
		c.logger.Info().Any("payload", event).Msg("object_update")

		_ = c.client.SendChangeEvent(event)

		c.metrics.ChangeEventCounter.With(
			map[string]string{
				EventTypeKey:  "object_update",
				ObjectKindKey: oldObject.GetKind(),
			},
		).Inc()
	}

}

func (c *ChangeCollector) addHandlerForGvr(gvr schema.GroupVersionResource, handler cache.ResourceEventHandler) {
	klog.Infof("starting to watch for resource %v", gvr)
	informer := c.informerFactory.ForResource(gvr)
	_, err := informer.Informer().AddEventHandler(handler)
	if err != nil {
		klog.Warningf("unable to watch for resource %v: %w", gvr, err)
	}
}

func (c *ChangeCollector) Start(ctx context.Context) error {
	klog.Infof("starting k8s resource collector process")

	handler := cache.ResourceEventHandlerFuncs{
		AddFunc:    c.OnAdd,
		UpdateFunc: c.OnUpdate,
		DeleteFunc: c.OnDelete,
	}

	allResources, err := GetAllResources(c.discoveryClient)
	if err != nil {
		return err
	}

	klog.Infof("all resources %v", allResources)
	for _, gvr := range WatchedGVRs {
		if _, ok := allResources[gvr]; ok {
			c.addHandlerForGvr(gvr, handler)
		} else {
			klog.Infof("skipping gvr %v", gvr)
		}
	}

	noOpHandler := cache.ResourceEventHandlerFuncs{
		AddFunc:    c.noOp,
		UpdateFunc: c.noOpUpdate,
		DeleteFunc: c.noOp,
	}

	c.addHandlerForGvr(eventGVR, noOpHandler) // only keep events in the cache, do not handle

	c.informerFactory.WaitForCacheSync(ctx.Done())
	c.informerFactory.Start(ctx.Done())
	c.startEventCollectionLoop(ctx)
	c.startBackupCollectionLoop(ctx)
	<-ctx.Done()
	klog.Infof("stopped k8s resource collector process")
	return nil
}

func (c *ChangeCollector) startEventCollectionLoop(ctx context.Context) {
	klog.Infof("starting to collect event resources every %v", c.eventCollectionInterval)

	go func() {
		for {
			select {
			case <-time.After(c.eventCollectionInterval):
				c.collectEvents()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (c *ChangeCollector) startBackupCollectionLoop(ctx context.Context) {
	klog.Infof("starting to collect workload resources every %v", c.backupCollectionInterval)

	go func() {
		for {
			select {
			case <-time.After(c.backupCollectionInterval):
				c.backupCollect()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (c *ChangeCollector) collectEvents() {
	klog.Infof("listing all resources for %v", eventGVR)
	listResult, err := c.informerFactory.ForResource(eventGVR).Lister().List(labels.Everything())
	if err != nil {
		klog.Error(err)
		return
	}

	if len(listResult) > 0 {
		c.logger.Info().Any("payload", listResult).Msg("resource_list")
		_ = c.client.SendK8sResources(api.NewResourceList(listResult))
	} else {
		klog.Infof("no result for %v", eventGVR)
	}
}

func (c *ChangeCollector) backupCollect() {
	for _, gvr := range BackupGVRs {
		klog.Infof("listing all resources for %v", gvr)
		listResult, err := c.informerFactory.ForResource(gvr).Lister().List(labels.Everything())
		if err != nil {
			klog.Error(err)
			return
		}

		if len(listResult) > 0 {
			c.logger.Info().Any("payload", listResult).Msg("resource_list")
			_ = c.client.SendK8sResources(api.NewResourceList(listResult))
		} else {
			klog.Infof("no result for %v", gvr)
		}
	}
}

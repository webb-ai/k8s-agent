package k8s

import (
	"context"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"time"
)

type Collector struct {
	DefaultResyncPeriod time.Duration
	DynamicClient       dynamic.Interface

	informerFactory dynamicinformer.DynamicSharedInformerFactory
}

func NewCollector(
	defaultResyncPeriod time.Duration,
	dynamicClient dynamic.Interface,
) *Collector {
	return &Collector{
		DefaultResyncPeriod: defaultResyncPeriod,
		DynamicClient:       dynamicClient,
	}
}

func (c *Collector) OnAdd(obj interface{}) {
	// TODO: retry on retryable errors
	klog.Infof("called on add %v", obj)
}

func (c *Collector) OnUpdate(oldObj, newObj interface{}) {
	klog.Infof("called on update %v", newObj)
}

func (c *Collector) OnDelete(obj interface{}) {
	klog.Infof("called on delete %v", obj)
}

func (c *Collector) Start(ctx context.Context) error {
	klog.Infof("starting k8s resource collector process")

	c.informerFactory = dynamicinformer.NewDynamicSharedInformerFactory(c.DynamicClient, c.DefaultResyncPeriod)

	eventHandler := cache.ResourceEventHandlerFuncs{
		AddFunc:    c.OnAdd,
		UpdateFunc: c.OnUpdate,
		DeleteFunc: c.OnDelete,
	}

	for _, gvr := range WatchedGVRs {
		klog.Infof("starting to watch for resource %v", gvr)
		informer := c.informerFactory.ForResource(gvr)
		informer.Informer().AddEventHandler(eventHandler)
	}
	c.informerFactory.WaitForCacheSync(ctx.Done())
	c.informerFactory.Start(ctx.Done())
	<-ctx.Done()
	klog.Infof("stopped k8s resource collector process")
	return nil
}

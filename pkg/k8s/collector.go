package k8s

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

type Collector struct {
	DefaultResyncPeriod time.Duration
	DynamicClient       dynamic.Interface

	logger          zerolog.Logger
	informerFactory dynamicinformer.DynamicSharedInformerFactory
}

type ResourceChangeEvent struct {
	OldObject runtime.Object `json:"oldObject"`
	NewObject runtime.Object `json:"newObject"`
}

func NewCollector(
	defaultResyncPeriod time.Duration,
	dynamicClient dynamic.Interface,
	logger zerolog.Logger,
) *Collector {
	return &Collector{
		DefaultResyncPeriod: defaultResyncPeriod,
		DynamicClient:       dynamicClient,
		logger:              logger,
	}
}

func (c *Collector) OnAdd(obj interface{}) {
	// TODO: retry on retryable errors
	runtimeObject, ok := obj.(runtime.Object)
	if !ok {
		return
	}

	c.logger.Info().
		Any("payload", ResourceChangeEvent{NewObject: runtimeObject}).Msg("object_add")
}

func (c *Collector) OnUpdate(oldObj, newObj interface{}) {
	oldRuntimeObj, ok := oldObj.(runtime.Object)
	if !ok {
		return
	}

	newRuntimeObj, ok := newObj.(runtime.Object)
	if !ok {
		return
	}

	c.logger.Info().
		Any("payload", ResourceChangeEvent{
			OldObject: oldRuntimeObj,
			NewObject: newRuntimeObj,
		}).Msg("object_update")
}

func (c *Collector) OnDelete(obj interface{}) {
	runtimeObject, ok := obj.(runtime.Object)
	if !ok {
		return
	}

	c.logger.Info().
		Any("payload", ResourceChangeEvent{OldObject: runtimeObject}).Msg("object_delete")
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

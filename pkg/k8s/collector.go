package k8s

import (
	"context"
	"time"

	"github.com/webb-ai/k8s-agent/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rs/zerolog"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

type Collector struct {
	DefaultResyncPeriod      time.Duration
	ResourceCollectionPeriod time.Duration
	DynamicClient            dynamic.Interface

	logger          zerolog.Logger
	informerFactory dynamicinformer.DynamicSharedInformerFactory
}

func NewCollector(
	defaultResyncPeriod time.Duration,
	resourceCollectionPeriod time.Duration,
	dynamicClient dynamic.Interface,
	logger zerolog.Logger,
) *Collector {
	return &Collector{
		DefaultResyncPeriod:      defaultResyncPeriod,
		ResourceCollectionPeriod: resourceCollectionPeriod,
		DynamicClient:            dynamicClient,
		logger:                   logger,
	}
}

func (c *Collector) OnAdd(obj interface{}) {
	// TODO: retry on retryable errors
	runtimeObject, err := interfacetoUnstructured(obj)
	if err != nil {
		klog.Error(err)
		return
	}

	event := api.NewResourceChangeEvent(nil, runtimeObject)
	c.logger.Info().Any("payload", event).Msg("object_add")
}

func (c *Collector) OnUpdate(oldObj, newObj interface{}) {
	oldObject, err := interfacetoUnstructured(oldObj)
	if err != nil {
		klog.Error(err)
		return
	}

	newObject, err := interfacetoUnstructured(newObj)
	if err != nil {
		klog.Error(err)
		return
	}

	event := api.NewResourceChangeEvent(oldObject, newObject)

	if oldObject.GetResourceVersion() != newObject.GetResourceVersion() {
		klog.Infof("detected resource version change of object")
		c.logger.Info().Any("payload", event).Msg("object_update")
	} else if hasStatusChanged(oldObject, newObject) {
		klog.Infof("detected status change of object")
		c.logger.Info().Any("payload", event).Msg("object_update")
	}

}

func (c *Collector) OnDelete(obj interface{}) {
	runtimeObject, err := interfacetoUnstructured(obj)
	if err != nil {
		klog.Error(err)
		return
	}

	event := api.NewResourceChangeEvent(runtimeObject, nil)

	c.logger.Info().Any("payload", event).Msg("object_delete")
}

func (c *Collector) Start(ctx context.Context) error {
	klog.Infof("starting k8s resource collector process")
	c.startWorkloadCollectionLoop(ctx)

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

func (c *Collector) startWorkloadCollectionLoop(ctx context.Context) {
	klog.Infof("starting to collect workload resources every %v", c.ResourceCollectionPeriod)

	go func() {
		for {
			select {
			case <-time.After(c.ResourceCollectionPeriod):
				c.collectWorkloadResources(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (c *Collector) collectWorkloadResources(ctx context.Context) {
	for _, gvr := range WorkloadGVRs {
		klog.Infof("listing all resources for %v", gvr)
		listResult, err := c.DynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
		if err != nil {
			klog.Error(err)
		} else {
			c.logger.Info().Any("payload", listResult.Items).Msg("resource_list")
		}
	}
}

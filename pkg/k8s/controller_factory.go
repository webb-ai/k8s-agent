package k8s

import (
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ControllerInitFunc func(gvk schema.GroupVersionKind) error

type ControllerFactory struct {
	controllerInitFunc ControllerInitFunc
	lock               *sync.RWMutex
	addedControllers   map[schema.GroupVersionKind]struct{}
}

// AddControllerForGvk adds a controller for GVK with factory's ControllerInitFunc
// it's up to the client to make sure the gvk is registered with workload cluster
func (factory *ControllerFactory) AddControllerForGvk(gvk schema.GroupVersionKind) error {
	factory.lock.Lock()
	defer factory.lock.Unlock()

	_, found := factory.addedControllers[gvk]
	if !found {
		err := factory.controllerInitFunc(gvk)
		if err == nil {
			factory.addedControllers[gvk] = struct{}{}
		}
		return err
	}
	return nil
}

func (factory *ControllerFactory) DoesControllerExistForGvk(gvk schema.GroupVersionKind) bool {
	factory.lock.RLock()
	defer factory.lock.RUnlock()

	_, found := factory.addedControllers[gvk]

	return found
}

func NewControllerFactory(controllerInitFunc ControllerInitFunc) *ControllerFactory {
	return &ControllerFactory{
		lock:               &sync.RWMutex{},
		controllerInitFunc: controllerInitFunc,
		addedControllers:   map[schema.GroupVersionKind]struct{}{},
	}
}

// func newFakeControllerFactory() *ControllerFactory {
// 	return NewControllerFactory(func(gvk schema.GroupVersionKind) error {
// 		return nil
//	})
// }

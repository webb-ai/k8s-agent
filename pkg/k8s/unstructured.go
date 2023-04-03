package k8s

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// runtimeObjecttoUnstructured converts the runtime object to unstructured
func runtimeObjecttoUnstructured(obj runtime.Object) (*unstructured.Unstructured, error) {
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)

	if err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{Object: unstructuredObj}, nil
}

// runtimeObjecttoUnstructured converts the runtime object to unstructured
func interfacetoUnstructured(obj interface{}) (*unstructured.Unstructured, error) {
	runtimeObj, ok := obj.(runtime.Object)
	if !ok {
		return nil, fmt.Errorf("%s isn't a k8s runtime object", obj)
	}
	return runtimeObjecttoUnstructured(runtimeObj)
}

// hasStatusChanged checks whether the status has changed from oldObject to newObject
func hasStatusChanged(oldObject *unstructured.Unstructured, newObject *unstructured.Unstructured) bool {
	oldMap, inOld, err := unstructured.NestedMap(oldObject.Object, "status")
	if err != nil {
		return false
	}
	newMap, inNew, err := unstructured.NestedMap(newObject.Object, "status")
	if err != nil {
		return false
	}

	if (!inOld && inNew) || (!inNew && inOld) {
		return true
	}

	return !reflect.DeepEqual(oldMap, newMap)
}

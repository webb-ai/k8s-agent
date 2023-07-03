package util

import (
	"fmt"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"

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

// UnstructuredToPod converts the unstructured content to Pod
func UnstructuredToPod(unstr *unstructured.Unstructured) (*corev1.Pod, error) {
	var pod corev1.Pod

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstr.UnstructuredContent(), &pod)
	if err != nil {
		return nil, err
	}
	return &pod, nil
}

// runtimeObjecttoUnstructured converts the runtime object to unstructured
func InterfaceToUnstructured(obj interface{}) (*unstructured.Unstructured, error) {
	runtimeObj, ok := obj.(runtime.Object)
	if !ok {
		return nil, fmt.Errorf("%s isn't a k8s runtime object", obj)
	}
	return runtimeObjecttoUnstructured(runtimeObj)
}

// hasStatusChanged checks whether the status has changed from oldObject to newObject
func HasStatusChanged(oldObject *unstructured.Unstructured, newObject *unstructured.Unstructured) bool {
	return hasFieldChanged(oldObject, newObject, "status")
}

func PruneData(object *unstructured.Unstructured) {
	if IsConfigMapOrSecret(object) {
		unstructured.RemoveNestedField(object.Object, "data")
	}
}

func IsConfigMapOrSecret(object *unstructured.Unstructured) bool {
	return object != nil && (object.GetKind() == "ConfigMap" || object.GetKind() == "Secret")
}

func HasDataChanged(oldObject, newObject *unstructured.Unstructured) bool {
	return hasFieldChanged(oldObject, newObject, "data")
}

func hasFieldChanged(oldObject, newObject *unstructured.Unstructured, field string) bool {
	oldMap, inOld, err := unstructured.NestedMap(oldObject.Object, field)
	if err != nil {
		return false
	}
	newMap, inNew, err := unstructured.NestedMap(newObject.Object, field)
	if err != nil {
		return false
	}

	if (!inOld && inNew) || (!inNew && inOld) {
		return true
	}

	return !reflect.DeepEqual(oldMap, newMap)
}

func GetCreationTimestamp(object *unstructured.Unstructured) (time.Time, error) {
	val, _, _ := unstructured.NestedString(object.Object, "metadata", "creationTimestamp")
	return time.Parse(time.RFC3339, val)
}

func GetDeletionTimestamp(object *unstructured.Unstructured) (time.Time, error) {
	val, _, _ := unstructured.NestedString(object.Object, "metadata", "deletionTimestamp")
	return time.Parse(time.RFC3339, val)
}

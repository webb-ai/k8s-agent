package util

import (
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	"reflect"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

// UnstructuredToDeployment converts the unstructured content to Deployment
func UnstructuredToDeployment(unstr *unstructured.Unstructured) (*appsv1.Deployment, error) {
	var deployment appsv1.Deployment

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstr.UnstructuredContent(), &deployment)
	if err != nil {
		return nil, err
	}
	return &deployment, nil
}

// UnstructuredToDaemonSet converts the unstructured content to DaemonSet
func UnstructuredToDaemonSet(unstr *unstructured.Unstructured) (*appsv1.DaemonSet, error) {
	var daemonset appsv1.DaemonSet

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstr.UnstructuredContent(), &daemonset)
	if err != nil {
		return nil, err
	}
	return &daemonset, nil
}

// UnstructuredToStatefulSet converts the unstructured content to StatefulSet
func UnstructuredToStatefulSet(unstr *unstructured.Unstructured) (*appsv1.StatefulSet, error) {
	var statefulset appsv1.StatefulSet

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstr.UnstructuredContent(), &statefulset)
	if err != nil {
		return nil, err
	}
	return &statefulset, nil
}

// UnstructuredToJob converts the unstructured content to Job
func UnstructuredToJob(unstr *unstructured.Unstructured) (*batchv1.Job, error) {
	var job batchv1.Job

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstr.UnstructuredContent(), &job)
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// UnstructuredToCronJob converts the unstructured content to CronJob
func UnstructuredToCronJob(unstr *unstructured.Unstructured) (*batchv1.CronJob, error) {
	var cronjob batchv1.CronJob

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstr.UnstructuredContent(), &cronjob)
	if err != nil {
		return nil, err
	}
	return &cronjob, nil
}

// UnstructuredToReplicaSet converts the unstructured content to ReplicaSet
func UnstructuredToReplicaSet(unstr *unstructured.Unstructured) (*appsv1.ReplicaSet, error) {
	var replicaset appsv1.ReplicaSet

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstr.UnstructuredContent(), &replicaset)
	if err != nil {
		return nil, err
	}
	return &replicaset, nil
}

// UnstructuredToService converts the unstructured content to Service
func UnstructuredToService(unstr *unstructured.Unstructured) (*corev1.Service, error) {
	var service corev1.Service

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstr.UnstructuredContent(), &service)
	if err != nil {
		return nil, err
	}
	return &service, nil
}

func GetLabelSelector(unstr *unstructured.Unstructured) (*metav1.LabelSelector, error) {
	selector, ok, err := unstructured.NestedFieldNoCopy(unstr.UnstructuredContent(), "spec", "selector")
	if !ok || err != nil {
		return nil, fmt.Errorf("unexpected error: data should have .spec.selector")
	}
	var labelSelector metav1.LabelSelector
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(
		selector.(map[string]interface{}), &labelSelector)
	if err != nil {
		return nil, err
	}
	return &labelSelector, nil
}

func GetClusterIP(unstr *unstructured.Unstructured) string {
	ip, ok, err := unstructured.NestedString(unstr.UnstructuredContent(), "spec", "clusterIP")
	if !ok || err != nil {
		return ""
	}
	return ip
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

func RedactEnvVar(object *unstructured.Unstructured) {
	if object == nil {
		return
	}
	// pods
	containers, ok, err := unstructured.NestedFieldNoCopy(object.UnstructuredContent(),
		"spec", "containers")
	if ok && err == nil {
		removeContainerEnv(containers)
	}

	// deployments, jobs, statefulsets, daemonsets
	containers, ok, err = unstructured.NestedFieldNoCopy(object.UnstructuredContent(),
		"spec", "template", "spec", "containers")
	if ok && err == nil {
		removeContainerEnv(containers)
	}

	// cronjobs
	containers, ok, err = unstructured.NestedFieldNoCopy(object.UnstructuredContent(),
		"spec", "jobTemplate", "spec", "template", "spec", "containers")
	if ok && err == nil {
		removeContainerEnv(containers)
	}

	// pods
	containers, ok, err = unstructured.NestedFieldNoCopy(object.UnstructuredContent(),
		"spec", "initContainers")
	if ok && err == nil {
		removeContainerEnv(containers)
	}

	// deployments, jobs, statefulsets, daemonsets

	containers, ok, err = unstructured.NestedFieldNoCopy(object.UnstructuredContent(),
		"spec", "template", "spec", "initContainers")
	if ok && err == nil {
		removeContainerEnv(containers)
	}

	// cronjobs

	containers, ok, err = unstructured.NestedFieldNoCopy(object.UnstructuredContent(),
		"spec", "jobTemplate", "spec", "template", "spec", "initContainers")
	if ok && err == nil {
		removeContainerEnv(containers)
	}
}

func removeContainerEnv(containers interface{}) {
	if containers == nil {
		return
	}

	containersSlice := containers.([]interface{})
	for _, container := range containersSlice {
		containerMap := container.(map[string]interface{})
		unstructured.RemoveNestedField(containerMap, "env")
	}

}

package api

import (
	"time"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/webb-ai/k8s-agent/pkg/util"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type EventType string

const (
	ObjectAdd    EventType = "object_add"
	ObjectUpdate EventType = "object_update"
	ObjectDelete EventType = "object_delete"
	KafkaUpdate  EventType = "kafka_update"
)

type ChangeEvent struct {
	OldObject *unstructured.Unstructured `json:"old_object"`
	NewObject *unstructured.Unstructured `json:"new_object"`
	EventType EventType                  `json:"event_type"`
	Time      int64                      `json:"time"`
}

func NewK8sChangeEvent(oldObj, newObj *unstructured.Unstructured) *ChangeEvent {
	util.PruneData(oldObj)
	util.PruneData(newObj)
	event := &ChangeEvent{
		OldObject: oldObj,
		NewObject: newObj,
		EventType: ObjectUpdate,
		Time:      time.Now().Unix(),
	}

	if oldObj == nil {
		event.EventType = ObjectAdd
	}
	if newObj == nil {
		event.EventType = ObjectDelete
	}
	return event
}

func NewKafkaChangeEvent(oldObj, newObj interface{}, apiKey string) *ChangeEvent {
	return &ChangeEvent{
		OldObject: &unstructured.Unstructured{Object: map[string]interface{}{apiKey: oldObj}},
		NewObject: &unstructured.Unstructured{Object: map[string]interface{}{apiKey: newObj}},
		EventType: KafkaUpdate,
		Time:      time.Now().Unix(),
	}
}

type ResourceList struct {
	Objects []runtime.Object `json:"objects"`
	Time    int64            `json:"time"`
}

func NewResourceList(objects []runtime.Object) *ResourceList {
	return &ResourceList{
		Objects: objects,
		Time:    time.Now().Unix(),
	}
}

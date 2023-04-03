package api

import (
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type EventType string

const (
	ObjectAdd    EventType = "object_add"
	ObjectUpdate EventType = "object_update"
	ObjectDelete EventType = "object_delete"
)

type ResourceChangeEvent struct {
	OldObject *unstructured.Unstructured `json:"oldObject"`
	NewObject *unstructured.Unstructured `json:"newObject"`
	EventType EventType                  `json:"eventType"`
	Time      int64                      `json:"time"`
}

func NewResourceChangeEvent(oldObj, newObj *unstructured.Unstructured) *ResourceChangeEvent {
	event := &ResourceChangeEvent{
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

type ResourceList struct {
	Objects []unstructured.Unstructured `json:"objects"`
}

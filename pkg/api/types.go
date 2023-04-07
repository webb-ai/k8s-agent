package api

import (
	"github.com/webb-ai/k8s-agent/pkg/util"
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
	OldObject *unstructured.Unstructured `json:"old_object"`
	NewObject *unstructured.Unstructured `json:"new_object"`
	EventType EventType                  `json:"event_type"`
	Time      int64                      `json:"time"`
}

func NewResourceChangeEvent(oldObj, newObj *unstructured.Unstructured) *ResourceChangeEvent {
	util.PruneData(oldObj)
	util.PruneData(newObj)
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
	Time    int64                       `json:"time"`
}

func NewResourceList(objects []unstructured.Unstructured) *ResourceList {
	return &ResourceList{
		Objects: objects,
		Time:    time.Now().Unix(),
	}
}

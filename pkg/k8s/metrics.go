package k8s

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const EventTypeKey = "event_type"
const ObjectKindKey = "object_kind"

type Metrics struct {
	ChangeEventCounter *prometheus.CounterVec
}

func NewMetrics() *Metrics {
	changeEventCounter := promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "change_event_total",
			Help: "Counts the total number of events received. Labels: event_type(object_add|object_update|object_delete)",
		},
		[]string{EventTypeKey, ObjectKindKey},
	)

	return &Metrics{
		ChangeEventCounter: changeEventCounter,
	}
}

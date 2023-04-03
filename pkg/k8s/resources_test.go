package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWatchedGVRs(t *testing.T) {
	t.Run("watched GVR should not have any duplicates", func(t *testing.T) {
		set := make(map[string]bool)

		for _, gvr := range WatchedGVRs {
			set[gvr.Group+gvr.Version+gvr.Resource] = true
		}

		assert.Equal(t, len(set), len(WatchedGVRs))
	})
}

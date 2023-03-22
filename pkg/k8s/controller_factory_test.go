package k8s

import (
	"fmt"

	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestControllerFactory(t *testing.T) {
	gvk := schema.GroupVersionKind{
		Group:   "test",
		Version: "test",
		Kind:    "test",
	}

	t.Run("good init func", func(t *testing.T) {
		initFunc := func(gvk schema.GroupVersionKind) error {
			return nil
		}

		factory := NewControllerFactory(initFunc)

		assert.False(t, factory.DoesControllerExistForGvk(gvk))
		err := factory.AddControllerForGvk(gvk)
		assert.NoError(t, err)
		assert.True(t, factory.DoesControllerExistForGvk(gvk))
	})

	t.Run("bad init func", func(t *testing.T) {
		testErr := fmt.Errorf("test error")
		initFunc := func(gvk schema.GroupVersionKind) error {
			return testErr
		}

		factory := NewControllerFactory(initFunc)

		assert.False(t, factory.DoesControllerExistForGvk(gvk))
		err := factory.AddControllerForGvk(gvk)
		assert.Equal(t, err, testErr)
		assert.False(t, factory.DoesControllerExistForGvk(gvk))
	})

}

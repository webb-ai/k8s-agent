package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestGetTimestamp(t *testing.T) {
	timeNow := metav1.Now()
	testCases := []struct {
		name string
		pod  *corev1.Pod
		err  error
	}{
		{"good pod", &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "default",
				Name:              "nginx",
				CreationTimestamp: timeNow,
				DeletionTimestamp: &timeNow,
			},
		}, nil},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			content, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(testCase.pod)
			unstr := &unstructured.Unstructured{Object: content}
			deletionTime, ok := GetDeletionTimestamp(unstr)
			assert.NoError(t, ok)
			creatimeTime, ok := GetCreationTimestamp(unstr)
			assert.NoError(t, ok)
			assert.Equal(t, creatimeTime, deletionTime)
		})
	}
}

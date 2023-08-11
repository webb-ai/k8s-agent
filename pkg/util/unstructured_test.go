package util

import (
	"github.com/google/go-cmp/cmp"
	batchv1 "k8s.io/api/batch/v1"

	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"

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

func TestRedactEnvVar(t *testing.T) {
	cases := []struct {
		name     string
		input    *unstructured.Unstructured
		expected *unstructured.Unstructured
	}{
		{
			name: "Pod with containers",
			input: toUnstructured(t, &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "container1",
							Env: []corev1.EnvVar{
								{
									Name:  "VAR1",
									Value: "value1",
								},
							},
						},
					},
				},
			}),
			expected: toUnstructured(t, &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "container1",
						},
					},
				},
			}),
		},
		{
			name: "Pod with init containers",
			input: toUnstructured(t, &corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name: "init-container1",
							Env: []corev1.EnvVar{
								{
									Name:  "VAR1",
									Value: "value1",
								},
							},
						},
					},
				},
			}),
			expected: toUnstructured(t, &corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name: "init-container1",
						},
					},
				},
			}),
		},
		{
			name: "CronJob with init containers and containers",
			input: toUnstructured(t, &batchv1.CronJob{
				Spec: batchv1.CronJobSpec{
					JobTemplate: batchv1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									InitContainers: []corev1.Container{
										{
											Name: "init-container1",
											Env: []corev1.EnvVar{
												{
													Name:  "VAR1",
													Value: "value1",
												},
											},
										},
									},
									Containers: []corev1.Container{
										{
											Name: "container1",
											Env: []corev1.EnvVar{
												{
													Name:  "VAR2",
													Value: "value2",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}),
			expected: toUnstructured(t, &batchv1.CronJob{
				Spec: batchv1.CronJobSpec{
					JobTemplate: batchv1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									InitContainers: []corev1.Container{
										{
											Name: "init-container1",
										},
									},
									Containers: []corev1.Container{
										{
											Name: "container1",
										},
									},
								},
							},
						},
					},
				},
			}),
		},

		{
			name: "Deployment with containers",
			input: toUnstructured(t, &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: "container1",
									Env: []corev1.EnvVar{
										{
											Name:  "VAR1",
											Value: "value1",
										},
									},
								},
							},
						},
					},
				},
			}),
			expected: toUnstructured(t, &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: "container1",
								},
							},
						},
					},
				},
			}),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			RedactEnvVar(tc.input)
			if diff := cmp.Diff(tc.expected, tc.input); diff != "" {
				t.Errorf("RedactEnvVar mismatch (-expected +actual):\n%s", diff)
			}
		})
	}
}

func toUnstructured(t *testing.T, obj runtime.Object) *unstructured.Unstructured {
	t.Helper()
	unstr, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		t.Fatalf("Failed to convert to unstructured: %v", err)
	}
	return &unstructured.Unstructured{Object: unstr}
}

package controller

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func testGenerateControllerPod(controller bool, apiVersion, kind, name string) *corev1.Pod {
	var owner []metav1.OwnerReference
	if controller {
		owner = append(owner, metav1.OwnerReference{
			APIVersion:         apiVersion,
			Kind:               kind,
			Name:               name,
			UID:                "abcdef",
			Controller:         &controller,
			BlockOwnerDeletion: &controller,
		})
	}

	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "demo",
			Namespace:       metav1.NamespaceDefault,
			OwnerReferences: owner,
		},
	}
}

func TestGetPodOwnerController(t *testing.T) {
	podSpec := corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:  "nginx",
				Image: "nginx:1.0",
			},
		},
	}
	client := fake.NewSimpleClientset()

	type Test struct {
		name        string
		pod         *corev1.Pod
		expected    []OwnerController
		expectedErr error
		createFn    func(c kubernetes.Interface) error
	}

	for _, test := range []Test{
		{
			name:     "null controller",
			pod:      testGenerateControllerPod(false, "v1", "none", "none"),
			expected: nil,
			createFn: func(c kubernetes.Interface) error {
				return nil
			},
		},
		{
			name: "job",
			pod:  testGenerateControllerPod(true, "batch/v1", "Job", "bar"),
			expected: []OwnerController{
				{
					Name:   "bar",
					Labels: nil,
					Kind:   "Job",
					Ready:  "1/1",
					Images: []string{"nginx:1.0"},
				},
			},
			createFn: func(c kubernetes.Interface) error {
				var completions int32 = 1
				job := &batchv1.Job{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Job",
						APIVersion: "batch/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bar",
						Namespace: metav1.NamespaceDefault,
					},
					Spec: batchv1.JobSpec{
						Completions: &completions,
						Template: corev1.PodTemplateSpec{
							Spec: podSpec,
						},
					},
					Status: batchv1.JobStatus{
						Succeeded: 1,
					},
				}
				_, err := c.BatchV1().Jobs(metav1.NamespaceDefault).Create(context.TODO(), job, metav1.CreateOptions{})
				return err
			},
		},
		{
			name: "replicaset",
			pod:  testGenerateControllerPod(true, "apps/v1", "ReplicaSet", "bar"),
			expected: []OwnerController{
				{
					Name:   "bar",
					Labels: nil,
					Kind:   "ReplicaSet",
					Ready:  "1/2",
					Images: []string{"nginx:1.0"},
				},
			},
			createFn: func(c kubernetes.Interface) error {
				rs := &appsv1.ReplicaSet{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ReplicaSet",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bar",
						Namespace: metav1.NamespaceDefault,
					},
					Spec: appsv1.ReplicaSetSpec{
						Template: corev1.PodTemplateSpec{
							Spec: podSpec,
						},
					},
					Status: appsv1.ReplicaSetStatus{
						Replicas:          2,
						ReadyReplicas:     1,
						AvailableReplicas: 1,
					},
				}
				_, err := c.AppsV1().ReplicaSets(metav1.NamespaceDefault).Create(context.TODO(), rs, metav1.CreateOptions{})
				return err
			},
		},
		{
			name: "statefulset",
			pod:  testGenerateControllerPod(true, "apps/v1", "StatefulSet", "bar"),
			expected: []OwnerController{
				{
					Name:   "bar",
					Labels: nil,
					Kind:   "StatefulSet",
					Ready:  "2/2",
					Images: []string{"nginx:1.0"},
				},
			},
			createFn: func(c kubernetes.Interface) error {
				sts := &appsv1.StatefulSet{
					TypeMeta: metav1.TypeMeta{
						Kind:       "StatefulSet",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bar",
						Namespace: metav1.NamespaceDefault,
					},
					Spec: appsv1.StatefulSetSpec{
						Template: corev1.PodTemplateSpec{
							Spec: podSpec,
						},
					},
					Status: appsv1.StatefulSetStatus{
						Replicas:      2,
						ReadyReplicas: 2,
					},
				}
				_, err := c.AppsV1().StatefulSets(metav1.NamespaceDefault).Create(context.TODO(), sts, metav1.CreateOptions{})
				return err
			},
		},
		{
			name: "daemonset",
			pod:  testGenerateControllerPod(true, "apps/v1", "DaemonSet", "bar"),
			expected: []OwnerController{
				{
					Name:   "bar",
					Labels: nil,
					Kind:   "DaemonSet",
					Ready:  "1/1",
					Images: []string{"nginx:1.0"},
				},
			},
			createFn: func(c kubernetes.Interface) error {
				ds := &appsv1.DaemonSet{
					TypeMeta: metav1.TypeMeta{
						Kind:       "DaemonSet",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bar",
						Namespace: metav1.NamespaceDefault,
					},
					Spec: appsv1.DaemonSetSpec{
						Template: corev1.PodTemplateSpec{
							Spec: podSpec,
						},
					},
					Status: appsv1.DaemonSetStatus{
						DesiredNumberScheduled: 1,
						NumberReady:            1,
					},
				}
				_, err := c.AppsV1().DaemonSets(metav1.NamespaceDefault).Create(context.TODO(), ds, metav1.CreateOptions{})
				return err
			},
		},
		{
			name: "replicationcontroller",
			pod:  testGenerateControllerPod(true, "v1", "ReplicationController", "bar"),
			expected: []OwnerController{
				{
					Name:   "bar",
					Labels: nil,
					Kind:   "ReplicationController",
					Ready:  "1/1",
					Images: []string{"nginx:1.0"},
				},
			},
			createFn: func(c kubernetes.Interface) error {
				rc := &corev1.ReplicationController{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ReplicationController",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bar",
						Namespace: metav1.NamespaceDefault,
					},
					Spec: corev1.ReplicationControllerSpec{
						Template: &corev1.PodTemplateSpec{
							Spec: podSpec,
						},
					},
					Status: corev1.ReplicationControllerStatus{
						Replicas:      1,
						ReadyReplicas: 1,
					},
				}
				_, err := c.CoreV1().ReplicationControllers(metav1.NamespaceDefault).Create(context.TODO(), rc, metav1.CreateOptions{})
				return err
			},
		},
		{
			name: "pod",
			pod:  testGenerateControllerPod(true, "v1", "Pod", "bar"),
			expected: []OwnerController{
				{
					Name:   "bar",
					Labels: nil,
					Kind:   "Pod",
					Ready:  "1/1",
					Images: []string{"nginx:1.0"},
				},
			},
			createFn: func(c kubernetes.Interface) error {
				pod := &corev1.Pod{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ReplicationController",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bar",
						Namespace: metav1.NamespaceDefault,
					},
					Spec: podSpec,
					Status: corev1.PodStatus{
						ContainerStatuses: []corev1.ContainerStatus{
							{
								Name:  "demo",
								Ready: true,
							},
						},
					},
				}
				_, err := c.CoreV1().Pods(metav1.NamespaceDefault).Create(context.TODO(), pod, metav1.CreateOptions{})
				return err
			},
		},
		{
			name:        "unknown kind deployment",
			pod:         testGenerateControllerPod(true, "apps/v1", "Deployment", "bar"),
			expected:    nil,
			expectedErr: fmt.Errorf("unknown reference kind: Deployment"),
			createFn: func(c kubernetes.Interface) error {
				deploy := &appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bar",
						Namespace: metav1.NamespaceDefault,
					},
					Spec: appsv1.DeploymentSpec{
						Template: corev1.PodTemplateSpec{
							Spec: podSpec,
						},
					},
					Status: appsv1.DeploymentStatus{
						Replicas:            2,
						UpdatedReplicas:     2,
						ReadyReplicas:       1,
						AvailableReplicas:   1,
						UnavailableReplicas: 1,
					},
				}
				_, err := c.AppsV1().Deployments(metav1.NamespaceDefault).Create(context.TODO(), deploy, metav1.CreateOptions{})
				return err
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.createFn(client)
			assert.NoError(t, err)
			got, err := GetPodOwnerController(client, test.pod)
			if err == nil && test.expectedErr != nil {
				t.Fatalf("expected error %q, but got none", test.expectedErr)
			} else if err != nil && test.expectedErr == nil {
				t.Fatalf("expected error none, but got %q", err)
			}

			if !reflect.DeepEqual(test.expected, got) {
				t.Errorf("unexpected result:\nexpected=%#v\ngot=%#v", test.expected, got)
			}
		})
	}
}

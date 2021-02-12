package node

import (
	"testing"

	"github.com/pytimer/k8sutil/types"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

// generateTestOnePod
// requests: cpu=50m, memory=300Mi, nvidia.com/gpu=2
// limits:cpu=100m, memory=512Mi, nvidia.com/gpu=2
func generateTestOnePod(namespace, name, nodeName string, podStatus corev1.PodPhase) corev1.Pod {
	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{Kind: "Pods", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: corev1.PodSpec{
			NodeName: nodeName,
			InitContainers: []corev1.Container{
				{
					Name:  "foo",
					Image: "alpine",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("10m"),
							corev1.ResourceMemory: resource.MustParse("300Mi"),
						},
					},
				},
			},
			Containers: []corev1.Container{
				{
					Name:  "bar",
					Image: "alpine",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("50m"),
							corev1.ResourceMemory: resource.MustParse("256Mi"),
							types.ResourceGPU:     resource.MustParse("2"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("512Mi"),
							types.ResourceGPU:     resource.MustParse("2"),
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: podStatus,
		},
	}
	return pod
}

func generateTestPodsOfNode() []corev1.Pod {
	pods := make([]corev1.Pod, 0)
	pods = append(pods, generateTestOnePod(metav1.NamespaceDefault, "pod-1", "worker1", corev1.PodRunning))
	pods = append(pods, generateTestOnePod(metav1.NamespaceSystem, "pod-2", "worker1", corev1.PodPending))
	pods = append(pods, generateTestOnePod(metav1.NamespaceDefault, "pod-3", "worker1", corev1.PodSucceeded))
	pods = append(pods, generateTestOnePod(metav1.NamespaceDefault, "pod-4", "worker1", corev1.PodFailed))
	pods = append(pods, generateTestOnePod(metav1.NamespaceDefault, "pod-5", "master", corev1.PodRunning))
	return pods
}

func TestGetPodsOfNode(t *testing.T) {
	pods := generateTestPodsOfNode()
	nodeName := "worker1"
	k8sClient := fake.Clientset{}
	k8sClient.AddReactor("*", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		switch action.(type) {
		case k8stesting.ListAction:
			podList := &corev1.PodList{}
			for i := range pods {
				if pods[i].Spec.NodeName == nodeName {
					podList.Items = append(podList.Items, pods[i])
				}
			}
			return true, podList, nil
		}
		return false, nil, nil
	})

	expected := 4
	podList, err := GetPodsOfNode(&k8sClient, nodeName)
	assert.NoError(t, err)
	assert.Equal(t, expected, len(podList.Items))
}

func TestGetNonTerminatedPodsOfNode(t *testing.T) {
	pods := generateTestPodsOfNode()
	nodeName := "worker1"
	k8sClient := fake.Clientset{}
	k8sClient.AddReactor("*", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		switch action.(type) {
		case k8stesting.ListAction:
			podList := &corev1.PodList{}
			for i := range pods {
				if pods[i].Spec.NodeName == nodeName {
					if pods[i].Status.Phase != corev1.PodSucceeded && pods[i].Status.Phase != corev1.PodFailed {
						podList.Items = append(podList.Items, pods[i])
					}
				}
			}
			return true, podList, nil
		}
		return false, nil, nil
	})

	expected := 2
	podList, err := GetNonTerminatedPodsOfNode(&k8sClient, nodeName)
	assert.NoError(t, err)
	assert.Equal(t, expected, len(podList.Items))
}

func TestGetNodeReadyStatus(t *testing.T) {
	type Test struct {
		name     string
		node     corev1.Node
		expected corev1.ConditionStatus
	}

	for _, test := range []Test{
		{
			name: "ready",
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "worker"},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
						{
							Type:   corev1.NodeMemoryPressure,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
			expected: corev1.ConditionTrue,
		},
		{
			name: "unknown",
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "worker"},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeMemoryPressure,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
			expected: corev1.ConditionUnknown,
		},
		{
			name: "not ready",
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "worker"},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionFalse,
						},
						{
							Type:   corev1.NodeMemoryPressure,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
			expected: corev1.ConditionFalse,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			actual := GetNodeReadyStatus(test.node)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestIsControlPlaneRole(t *testing.T) {
	type Test struct {
		name     string
		node     corev1.Node
		expected bool
	}
	for _, test := range []Test{
		{
			name: "node-role.kubernetes.io/master",
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"node-role.kubernetes.io/master": "",
					},
				},
			},
			expected: true,
		},
		{
			name: "node-role.kubernetes.io/not master",
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"node-role.kubernetes.io/worker": "",
					},
				},
			},
			expected: false,
		},
		{
			name: "kubernetes.io/role=master",
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"kubernetes.io/role": "master",
					},
				},
			},
			expected: true,
		},
		{
			name: "kubernetes.io/role not master",
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"kubernetes.io/role": "worker",
					},
				},
			},
			expected: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			actual := IsControlPlaneRole(&test.node)
			assert.Equal(t, test.expected, actual)
		})
	}
}

package node

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	intypes "github.com/pytimer/k8sutil/types"
	"github.com/pytimer/k8sutil/util"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes"
)

func GetNonTerminatedPodsOfNode(client kubernetes.Interface, nodeName string) (*corev1.PodList, error) {
	selector := fmt.Sprintf("spec.nodeName=%s,status.phase!=%s,status.phase!=%s", nodeName, string(corev1.PodSucceeded), string(corev1.PodFailed))
	fieldSelector, err := fields.ParseSelector(selector)
	if err != nil {
		return nil, err
	}
	pods, err := client.CoreV1().Pods(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{
		FieldSelector: fieldSelector.String(),
	})
	if err != nil {
		return nil, err
	}
	return pods, nil
}

func GetPodsOfNode(client kubernetes.Interface, nodeName string) (*corev1.PodList, error) {
	fieldSelector, err := fields.ParseSelector(fmt.Sprintf("spec.nodeName=%s", nodeName))
	if err != nil {
		return nil, err
	}
	return client.CoreV1().Pods(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{
		FieldSelector: fieldSelector.String(),
	})
}

func GetNodeReadyStatus(node corev1.Node) corev1.ConditionStatus {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status
		}
	}
	return corev1.ConditionUnknown
}

// findNodeRoles returns the roles of a given node.
// The roles are determined by looking for:
//   node-role.kubernetes.io/<role>=""
//   kubernetes.io/role="<role>"
func findNodeRoles(node *corev1.Node) []string {
	roles := sets.NewString()
	for k, v := range node.Labels {
		switch {
		case strings.HasPrefix(k, intypes.LabelNodeRolePrefix):
			if role := strings.TrimPrefix(k, intypes.LabelNodeRolePrefix); len(role) > 0 {
				roles.Insert(role)
			}
		case k == intypes.NodeLabelRole && v != "":
			roles.Insert(v)
		}
	}
	return roles.List()
}

func IsControlPlaneRole(node *corev1.Node) bool {
	roles := findNodeRoles(node)
	for _, v := range roles {
		if strings.ToLower(v) == intypes.NodeRoleControlPlane ||
			strings.ToLower(v) == intypes.NodeRoleMaster {
			return true
		}
	}
	return false
}

// Patch patch node
func Patch(ctx context.Context, c kubernetes.Interface, node *corev1.Node, patchFn func(*corev1.Node), opts metav1.PatchOptions) (*corev1.Node, error) {
	oldData, err := json.Marshal(node)
	if err != nil {
		return nil, err
	}

	patchFn(node)

	newData, err := json.Marshal(node)
	if err != nil {
		return nil, err
	}

	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, corev1.Node{})
	if err != nil {
		return nil, errors.Wrapf(err, "unable create two way merge patch bytes")
	}
	return c.CoreV1().Nodes().Patch(ctx, node.Name, types.StrategicMergePatchType, patchBytes, opts)
}

func PatchNodeLabel(c kubernetes.Interface, node *corev1.Node, labels map[string]string) (*corev1.Node, error) {
	oldData, err := json.Marshal(node)
	if err != nil {
		return nil, err
	}

	newLabels := util.MergeStringMaps(node.Labels, labels)
	node.SetLabels(newLabels)
	newData, err := json.Marshal(node)
	if err != nil {
		return nil, err
	}

	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, corev1.Node{})
	if err != nil {
		return nil, err
	}

	opts := metav1.PatchOptions{}
	return c.CoreV1().Nodes().Patch(context.Background(), node.Name, types.StrategicMergePatchType, patchBytes, opts)
}

package history

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/klog/v2"
)

const (
	DeploymentRevisionAnnotation = "deployment.kubernetes.io/revision"
)

type DeploymentViewer struct {
	c kubernetes.Interface
}

func (h *DeploymentViewer) ViewHistory(namespace, name string) ([]*RevisionHistory, []*RevisionHistory, error) {
	appsClient := h.c.AppsV1()
	deployment, err := appsClient.Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}

	rsList, err := listReplicaSets(appsClient, deployment)
	if err != nil {
		return nil, nil, err
	}

	newRS := findNewReplicaSet(deployment, rsList)
	oldRSs := findOldReplicaSets(rsList, newRS)

	newRH, err := singleReplicaSetToRevisionHistory(newRS)
	if err != nil {
		return nil, nil, err
	}

	oldRHs := replicaSetsToRevisionHistory(oldRSs)

	return []*RevisionHistory{&newRH}, oldRHs, nil
}

func replicaSetsToRevisionHistory(rsList []*appsv1.ReplicaSet) []*RevisionHistory {
	rhs := make([]*RevisionHistory, 0, len(rsList))
	for _, rs := range rsList {
		rh, err := singleReplicaSetToRevisionHistory(rs)
		if err != nil {
			klog.Warningf("ReplicaSet %s to revision history fail, %v", rs.Name, err)
			continue
		}
		rhs = append(rhs, &rh)
	}
	return rhs
}

func singleReplicaSetToRevisionHistory(rs *appsv1.ReplicaSet) (RevisionHistory, error) {
	revision, err := Revision(rs)
	if err != nil {
		return RevisionHistory{}, err
	}

	status := HistoryStatus{
		DesiredReplicas:      rs.Status.Replicas,
		FullyLabeledReplicas: rs.Status.FullyLabeledReplicas,
		ReadyReplicas:        rs.Status.ReadyReplicas,
		AvailableReplicas:    rs.Status.AvailableReplicas,
	}

	images := make([]string, 0)
	for _, c := range rs.Spec.Template.Spec.InitContainers {
		images = append(images, c.Image)
	}

	for _, c := range rs.Spec.Template.Spec.Containers {
		images = append(images, c.Image)
	}

	return RevisionHistory{
		Revision:        revision,
		Name:            rs.Name,
		Labels:          rs.Labels,
		Annotations:     rs.Annotations,
		CreateTimestamp: rs.CreationTimestamp,
		Status:          status,
		Images:          images,
	}, nil
}

func Revision(obj runtime.Object) (int64, error) {
	acc, err := meta.Accessor(obj)
	if err != nil {
		return 0, err
	}
	v, ok := acc.GetAnnotations()[DeploymentRevisionAnnotation]
	if !ok {
		return 0, nil
	}
	return strconv.ParseInt(v, 10, 64)
}

// equalIgnoreHash 如果deployment和replicasSet的podTemplateSpec相同，返回true。
// 对比的过程中忽略pod-template-hash，因为deployment没有该字段。
func equalIgnoreHash(template1, template2 *corev1.PodTemplateSpec) bool {
	t1 := template1.DeepCopy()
	t2 := template2.DeepCopy()
	delete(t1.Labels, appsv1.DefaultDeploymentUniqueLabelKey)
	delete(t2.Labels, appsv1.DefaultDeploymentUniqueLabelKey)
	return apiequality.Semantic.DeepEqual(t1, t2)
}

func findNewReplicaSet(deployment *appsv1.Deployment, rsList []*appsv1.ReplicaSet) *appsv1.ReplicaSet {
	sort.Sort(replicaSetsByCreationTimestamp(rsList))
	for _, r := range rsList {
		if equalIgnoreHash(&deployment.Spec.Template, &r.Spec.Template) {
			return r
		}
	}
	return nil
}

func findOldReplicaSets(rsList []*appsv1.ReplicaSet, newRs *appsv1.ReplicaSet) []*appsv1.ReplicaSet {
	var oldRSs []*appsv1.ReplicaSet
	for _, rs := range rsList {
		if newRs != nil && newRs.UID == rs.UID {
			continue
		}
		oldRSs = append(oldRSs, rs)
	}
	return oldRSs
}

type replicaSetsByCreationTimestamp []*appsv1.ReplicaSet

func (r replicaSetsByCreationTimestamp) Len() int {
	return len(r)
}

func (r replicaSetsByCreationTimestamp) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r replicaSetsByCreationTimestamp) Less(i, j int) bool {
	if r[i].CreationTimestamp.Equal(&r[j].CreationTimestamp) {
		return r[i].Name < r[j].Name
	}
	return r[i].CreationTimestamp.Before(&r[j].CreationTimestamp)
}

func listReplicaSets(appsClient appsv1client.AppsV1Interface, deployment *appsv1.Deployment) ([]*appsv1.ReplicaSet, error) {
	labelSelector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return nil, err
	}

	options := metav1.ListOptions{LabelSelector: labelSelector.String()}
	rsList, err := appsClient.ReplicaSets(deployment.Namespace).List(context.TODO(), options)
	if err != nil {
		return nil, err
	}

	list := rsList.Items
	// 获取属于该deployment的ReplicaSet
	owned := make([]*appsv1.ReplicaSet, 0, len(list))
	for i := range list {
		if metav1.IsControlledBy(&list[i], deployment) {
			owned = append(owned, &list[i])
		}
	}

	return owned, nil
}

type StatefulSetViewer struct {
	c kubernetes.Interface
}

func (h *StatefulSetViewer) ViewHistory(namespace, name string) ([]*RevisionHistory, []*RevisionHistory, error) {
	return nil, nil, fmt.Errorf("not implement")
}

type DaemonsetViewer struct {
	c kubernetes.Interface
}

func (h *DaemonsetViewer) ViewHistory(namespace, name string) ([]*RevisionHistory, []*RevisionHistory, error) {
	return nil, nil, fmt.Errorf("not implement")
}

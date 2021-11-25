package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/pytimer/k8sutil/podutil"
	"github.com/pytimer/k8sutil/types"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

type OwnerController struct {
	Name              string            `json:"name"`
	Labels            map[string]string `json:"labels"`
	Kind              string            `json:"kind"`
	Ready             string            `json:"ready"`
	Images            []string          `json:"images"`
	CreationTimestamp metav1.Time       `json:"creationTimestamp"`
}

func GetPodOwnerController(client kubernetes.Interface, pod *corev1.Pod) ([]OwnerController, error) {
	ownerRef := metav1.GetControllerOf(pod)
	if ownerRef == nil {
		return nil, nil
	}

	oc := OwnerController{
		Name: ownerRef.Name,
		Kind: ownerRef.Kind,
	}
	switch strings.ToLower(ownerRef.Kind) {
	case types.ResourceKindJob:
		job, err := client.BatchV1().Jobs(pod.Namespace).Get(context.TODO(), ownerRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		oc.Labels = job.Labels
		oc.Ready = fmt.Sprintf("%d/%d", job.Status.Succeeded, *job.Spec.Completions)
		oc.Images = podutil.GetPodImages(job.Spec.Template.Spec)
		oc.CreationTimestamp = job.CreationTimestamp
	case types.ResourceKindReplicationController:
		rc, err := client.CoreV1().ReplicationControllers(pod.Namespace).Get(context.TODO(), ownerRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		oc.Labels = rc.Labels
		oc.Ready = fmt.Sprintf("%d/%d", rc.Status.ReadyReplicas, rc.Status.Replicas)
		oc.Images = podutil.GetPodImages(rc.Spec.Template.Spec)
		oc.CreationTimestamp = rc.CreationTimestamp
	case types.ResourceKindReplicaSet:
		rs, err := client.AppsV1().ReplicaSets(pod.Namespace).Get(context.TODO(), ownerRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		oc.Labels = rs.Labels
		oc.Ready = fmt.Sprintf("%d/%d", rs.Status.ReadyReplicas, rs.Status.Replicas)
		oc.Images = podutil.GetPodImages(rs.Spec.Template.Spec)
		oc.CreationTimestamp = rs.CreationTimestamp
	case types.ResourceKindDaemonSet:
		ds, err := client.AppsV1().DaemonSets(pod.Namespace).Get(context.TODO(), ownerRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		oc.Labels = ds.Labels
		oc.Ready = fmt.Sprintf("%d/%d", ds.Status.NumberReady, ds.Status.DesiredNumberScheduled)
		oc.Images = podutil.GetPodImages(ds.Spec.Template.Spec)
		oc.CreationTimestamp = ds.CreationTimestamp
	case types.ResourceKindStatefulSet:
		sts, err := client.AppsV1().StatefulSets(pod.Namespace).Get(context.TODO(), ownerRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		oc.Labels = sts.Labels
		oc.Ready = fmt.Sprintf("%d/%d", sts.Status.ReadyReplicas, sts.Status.Replicas)
		oc.Images = podutil.GetPodImages(sts.Spec.Template.Spec)
		oc.CreationTimestamp = sts.CreationTimestamp
	case types.ResourceKindPod:
		p, err := client.CoreV1().Pods(pod.Namespace).Get(context.TODO(), ownerRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		oc.Labels = p.Labels
		oc.Ready = fmt.Sprintf("%d/%d", len(podutil.GetPodReadyContainers(p)), len(p.Spec.Containers))
		oc.Images = podutil.GetPodImages(p.Spec)
		oc.CreationTimestamp = p.CreationTimestamp
	default:
		return nil, fmt.Errorf("unknown reference kind: %s", ownerRef.Kind)

	}
	klog.V(5).Infof("owner controller: %#v", oc)
	return []OwnerController{oc}, nil
}

// GetPodsOfController return pods created by controller name
func GetPodsOfController(client kubernetes.Interface, namespace, kind, name, apiVersion string) ([]corev1.Pod, error) {
	type controller struct {
		kind       string
		apiVersion string
		Selector   *metav1.LabelSelector
	}

	ct := controller{}
	switch strings.ToLower(kind) {
	case types.ResourcePluralKindJob:
		job, err := client.BatchV1().Jobs(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		ct.kind = types.ResourceKindJob
		ct.apiVersion = "batch/v1"
		ct.Selector = job.Spec.Selector
	case types.ResourcePluralKindReplicationController:
		rc, err := client.CoreV1().ReplicationControllers(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		ct.kind = types.ResourceKindReplicationController
		ct.apiVersion = "v1"
		ct.Selector = &metav1.LabelSelector{
			MatchLabels: rc.Spec.Selector,
		}
	case types.ResourcePluralKindReplicaSet:
		rs, err := client.AppsV1().ReplicaSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		ct.kind = types.ResourceKindReplicaSet
		ct.apiVersion = "apps/v1"
		ct.Selector = rs.Spec.Selector
	case types.ResourcePluralKindDaemonSet:
		ds, err := client.AppsV1().DaemonSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		ct.kind = types.ResourceKindDaemonSet
		ct.apiVersion = "apps/v1"
		ct.Selector = ds.Spec.Selector
	case types.ResourcePluralKindStatefulSet:
		sts, err := client.AppsV1().StatefulSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		ct.kind = types.ResourceKindStatefulSet
		ct.apiVersion = "apps/v1"
		ct.Selector = sts.Spec.Selector
	case types.ResourcePluralKindService:
		srv, err := client.CoreV1().Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		ct.kind = types.ResourceKindService
		ct.apiVersion = "v1"
		ct.Selector = &metav1.LabelSelector{
			MatchLabels: srv.Spec.Selector,
		}
	default:
		return nil, fmt.Errorf("unknown reference kind: %s", kind)
	}

	klog.V(5).Infof("resource kind: %#v", ct)

	labelSelector, err := metav1.LabelSelectorAsSelector(ct.Selector)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse spec.selector")
	}
	klog.V(5).Infof("label selector %v", labelSelector)
	opts := metav1.ListOptions{LabelSelector: labelSelector.String()}
	podList, err := client.CoreV1().Pods(namespace).List(context.TODO(), opts)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get pods")
	}
	klog.V(10).Infof("podList: %v", podList)
	if ct.kind == types.ResourceKindService {
		return podList.Items, nil
	}

	pods := make([]corev1.Pod, 0, len(podList.Items))
	for _, pod := range podList.Items {
		ownerRef := metav1.GetControllerOf(&pod)
		if ownerRef == nil {
			continue
		}
		klog.V(5).Infof("pod ownerRef: %#v", ownerRef)
		if strings.ToLower(ownerRef.Kind) == ct.kind && ownerRef.Name == name && ownerRef.APIVersion == ct.apiVersion {
			pods = append(pods, pod)
		}
	}
	klog.V(5).Infof("podList filter: %v", pods)
	return pods, nil
}

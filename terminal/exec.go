package terminal

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

func ValidatePod(ctx context.Context, client kubernetes.Interface, namespace, podName, containerName string) error {
	pod, err := client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		return fmt.Errorf("pod %s/%s not found", namespace, podName)
	}

	if err != nil {
		return err
	}

	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		return fmt.Errorf("cannot exec into container in a completed pod, current phase %s", pod.Status.Phase)
	}

	for _, cc := range pod.Spec.InitContainers {
		if containerName == cc.Name {
			return fmt.Errorf("can't exec init container %s in pod %s/%s ", containerName, namespace, podName)
		}
	}

	for _, cs := range pod.Status.ContainerStatuses {
		if containerName == cs.Name {
			return nil
		}
	}

	return fmt.Errorf("pod has no container %s", containerName)
}

func (t *TerminalSession) Exec(config *rest.Config, namespace, podName, containerName string, cmd []string) error {
	req := t.client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec")

	req.VersionedParams(&corev1.PodExecOptions{
		Container: containerName,
		Command:   cmd,
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return err
	}

	return executor.Stream(remotecommand.StreamOptions{
		Stdin:             t,
		Stdout:            t,
		Stderr:            t,
		Tty:               true,
		TerminalSizeQueue: t,
	})
}

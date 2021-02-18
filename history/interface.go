package history

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

type Viewer interface {
	ViewHistory(namespace, name string) ([]*RevisionHistory, []*RevisionHistory, error)
}

func ViewerFor(c kubernetes.Interface, kind schema.GroupKind) Viewer {
	switch {
	case groupMatch(kind.Group, "apps", "extensions") && kind.Kind == "Deployment":
		return &DeploymentViewer{c}
	case groupMatch(kind.Group, "apps", "extensions") && kind.Kind == "StatefulSet":
		return &StatefulSetViewer{c}
	case groupMatch(kind.Group, "apps", "extensions") && kind.Kind == "Daemonset":
		return &DaemonsetViewer{c}
	}
	return nil
}

func groupMatch(group string, matches ...string) bool {
	for _, g := range matches {
		if group == g {
			return true
		}
	}
	return false
}

type RevisionHistory struct {
	Revision        int64             `json:"revision"`
	Name            string            `json:"name"`
	Labels          map[string]string `json:"labels"`
	Annotations     map[string]string `json:"annotations"`
	CreateTimestamp metav1.Time       `json:"createTimestamp"`
	Status          HistoryStatus     `json:"status"`
	Images          []string          `json:"images"`
}

type HistoryStatus struct {
	DesiredReplicas      int32 `json:"desiredReplicas"`
	FullyLabeledReplicas int32 `json:"fullyLabeledReplicas,omitempty"`
	ReadyReplicas        int32 `json:"readyReplicas"`
	AvailableReplicas    int32 `json:"availableReplicas" `
}

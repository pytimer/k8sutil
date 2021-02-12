package metrics

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsapi "k8s.io/metrics/pkg/apis/metrics"
	metricsV1beta1api "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	resourceclient "k8s.io/metrics/pkg/client/clientset/versioned"
)

type resourceMetricsClient struct {
	client resourceclient.Interface
}

func NewResourceMetricsClient(resourceClient resourceclient.Interface) *resourceMetricsClient {
	return &resourceMetricsClient{client: resourceClient}
}

func (c *resourceMetricsClient) GetNodeResourceMetric(ctx context.Context, name string) (*metricsapi.NodeMetrics, error) {
	versionedMetrics, err := c.client.MetricsV1beta1().NodeMetricses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	m := &metricsapi.NodeMetrics{}
	if err := metricsV1beta1api.Convert_v1beta1_NodeMetrics_To_metrics_NodeMetrics(versionedMetrics, m, nil); err != nil {
		return nil, err
	}

	return m, nil
}

package metrics

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/agoda-com/macOS-vz-kubelet/pkg/client"
	"github.com/agoda-com/macOS-vz-kubelet/pkg/metrics/collectors"

	"github.com/virtual-kubelet/virtual-kubelet/log"
	"github.com/virtual-kubelet/virtual-kubelet/trace"
	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"

	dto "github.com/prometheus/client_model/go"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	corev1listers "k8s.io/client-go/listers/core/v1"
	compbasemetrics "k8s.io/component-base/metrics"
)

type MacOSVZPodMetricsProvider struct {
	nodeName    string
	metricsSync sync.Mutex

	podLister corev1listers.PodLister
	vzClient  client.VzClientInterface
}

// NewMacOSVZPodMetricsProvider creates a new MacOSVZPodMetricsProvider
func NewMacOSVZPodMetricsProvider(nodeName string, podLister corev1listers.PodLister, vzClient client.VzClientInterface) *MacOSVZPodMetricsProvider {
	return &MacOSVZPodMetricsProvider{
		nodeName:  nodeName,
		podLister: podLister,
		vzClient:  vzClient,
	}
}

// GetStatsSummary gets the stats for the node, including running pods
func (p *MacOSVZPodMetricsProvider) GetStatsSummary(ctx context.Context) (summary *stats.Summary, err error) {
	ctx, span := trace.StartSpan(ctx, "MacOSVZPodMetricsProvider.GetStatsSummary")
	defer func() {
		span.SetStatus(err)
		span.End()
	}()
	log.G(ctx).Debug("Received GetStatsSummary request")

	p.metricsSync.Lock()
	defer p.metricsSync.Unlock()

	log.G(ctx).Debug("Acquired metrics lock")

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	pods, err := p.podLister.List(labels.Everything())
	if err != nil {
		log.G(ctx).WithError(err).Errorf("failed to retrieve pods list")
	}

	g := errgroup.Group{}
	results := make([]stats.PodStats, len(pods))

	for i, pod := range pods {
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}

		g.Go(func() (err error) {
			ctx, span := trace.StartSpan(ctx, "getPodMetrics")
			defer func() {
				span.SetStatus(err)
				span.End()
			}()
			ctx = span.WithFields(ctx, log.Fields{
				"UID":       string(pod.UID),
				"Name":      pod.Name,
				"Namespace": pod.Namespace,
			})

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// Get the virtualization group for the pod
			cs, err := p.vzClient.GetVirtualizationGroupStats(ctx, pod.Namespace, pod.Name, pod.Spec.Containers)
			if err != nil {
				return fmt.Errorf("failed to get virtualization group stats for pod %s/%s: %w", pod.Namespace, pod.Name, err)
			}

			results[i] = stats.PodStats{
				PodRef: stats.PodReference{
					Name:      pod.Name,
					Namespace: pod.Namespace,
					UID:       string(pod.UID),
				},
				StartTime:  pod.CreationTimestamp,
				Containers: cs,
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("failed to get stats for all pods: %w", err)
	}

	var s stats.Summary
	s.Node = stats.NodeStats{
		NodeName: p.nodeName,
	}
	s.Pods = results

	return &s, nil
}

// GetMetricsResource gets the metrics for the node, including running pods
func (p *MacOSVZPodMetricsProvider) GetMetricsResource(ctx context.Context) (mf []*dto.MetricFamily, err error) {
	ctx, span := trace.StartSpan(ctx, "MacOSVZPodMetricsProvider.GetMetricsResource")
	defer func() {
		span.SetStatus(err)
		span.End()
	}()
	log.G(ctx).Debug("Received GetMetricsResource request")

	statsSummary, err := p.GetStatsSummary(ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching MetricsResource: %w", err)
	}

	registry := compbasemetrics.NewKubeRegistry()
	registry.CustomMustRegister(collectors.NewKubeletResourceMetricsCollector(statsSummary))

	metricFamily, err := registry.Gather()
	if err != nil {
		return nil, fmt.Errorf("error gathering metrics: %w", err)
	}

	return metricFamily, nil
}

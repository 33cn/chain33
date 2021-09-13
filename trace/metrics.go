package trace

import (
	"github.com/33cn/chain33/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

func newMetricsRegistry() (r *prometheus.Registry) {
	r = prometheus.NewRegistry()
	// register standard metrics
	r.MustRegister(
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{
			Namespace: metrics.Namespace,
		}),
		collectors.NewGoCollector(),
		prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: metrics.Namespace,
			Name:      "info",
			Help:      "chain33 information.",
			ConstLabels: prometheus.Labels{
				"version": "",
			},
		}),
	)

	return r
}

func (s *Service) MustRegisterMetrics(cs ...prometheus.Collector) {
	s.metricsRegistry.MustRegister(cs...)
}

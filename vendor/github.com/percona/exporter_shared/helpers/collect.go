package helpers

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Collector is a subset of prometheus.Collector with a single method.
type Collector interface {
	Collect(ch chan<- prometheus.Metric)
}

// CollectMetrics receives all metrics from collector.
func CollectMetrics(collector Collector) []prometheus.Metric {
	ch := make(chan prometheus.Metric)
	go func() {
		collector.Collect(ch)
		close(ch)
	}()

	res := make([]prometheus.Metric, 0, 10)
	for m := range ch {
		res = append(res, m)
	}
	return res
}

// check interfaces
var (
	_ Collector = prometheus.Collector(nil)
)

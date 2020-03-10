// Copyright 2017 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helpers

import (
	"io"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

type collector struct {
	metrics []prometheus.Metric
}

func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range c.metrics {
		ch <- m.Desc()
	}
}

func (c *collector) Collect(ch chan<- prometheus.Metric) {
	for _, m := range c.metrics {
		ch <- m
	}
}

// Format converts a slice of Prometheus metrics to strings in text exposition format.
func Format(metrics []prometheus.Metric) []string {
	r := prometheus.NewPedanticRegistry()
	r.MustRegister(&collector{metrics: metrics})
	families, err := r.Gather()
	if err != nil {
		panic(err)
	}

	var buf strings.Builder
	e := expfmt.NewEncoder(&buf, expfmt.FmtText)
	for _, f := range families {
		if err = e.Encode(f); err != nil {
			panic(err)
		}
	}
	return strings.Split(strings.TrimSpace(buf.String()), "\n")
}

// Parse converts strings in text exposition format to a slice of Prometheus metrics.
func Parse(metrics []string) []prometheus.Metric {
	r := strings.NewReader(strings.Join(metrics, "\n") + "\n")
	d := expfmt.NewDecoder(r, expfmt.FmtText)

	res := make([]prometheus.Metric, 0, 10)
	for {
		var family dto.MetricFamily
		if err := d.Decode(&family); err != nil {
			if err == io.EOF {
				return res
			}
			panic(err)
		}

		for _, m := range family.Metric {
			labels, typ, value := readDTOMetric(m)
			mm := &Metric{family.GetName(), family.GetHelp(), labels, typ, value}
			res = append(res, mm.Metric())
		}
	}
}

// check interfaces
var (
	_ prometheus.Collector = (*collector)(nil)
)

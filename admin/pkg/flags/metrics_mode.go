// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package flags

import (
	"github.com/AlekSi/pointer"

	"github.com/percona/pmm/utils/enums"
)

// MetricsModeFlags contains flags for metrics mode.
type MetricsModeFlags struct {
	MetricsMode MetricsMode `enum:"${metricsModesEnum}" default:"auto" help:"Metrics flow mode, can be push - agent will push metrics, pull - server scrape metrics from agent or auto - chosen by server"` //nolint:lll
}

// MetricsMode is a structure for metrics mode flag.
type MetricsMode string

// EnumValue returns pointer to string representation of LogLevel.
func (l MetricsMode) EnumValue() *string {
	return pointer.To(enums.ConvertEnum("METRICS_MODE", string(l)))
}

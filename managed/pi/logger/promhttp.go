// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package logger

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// PromHTTP is a compatibility wrapper between zap's sugared logger entry
// and Prometheus HTTP logger interface.
type PromHTTP struct {
	L *zap.SugaredLogger
}

// Println prints log message with info level.
func (p *PromHTTP) Println(args ...interface{}) { p.L.Info(args...) }

// Check interfaces.
var _ promhttp.Logger = (*PromHTTP)(nil)

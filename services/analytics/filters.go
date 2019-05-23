// qan-api2
// Copyright (C) 2019 Percona LLC
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

package analitycs

import (
	"context"
	"fmt"

	"github.com/percona/pmm/api/qanpb"
)

// Get implements rpc to get list of available labels.
func (s *Service) Get(ctx context.Context, in *qanpb.FiltersRequest) (*qanpb.FiltersReply, error) {

	if in.PeriodStartFrom == nil || in.PeriodStartTo == nil {
		err := fmt.Errorf("from-date: %s or to-date: %s cannot be empty", in.PeriodStartFrom, in.PeriodStartTo)
		return nil, err
	}

	periodStartFromSec := in.PeriodStartFrom.Seconds
	periodStartToSec := in.PeriodStartTo.Seconds
	if periodStartFromSec > periodStartToSec {
		err := fmt.Errorf("from-date %s cannot be bigger then to-date %s", in.PeriodStartFrom, in.PeriodStartTo)
		return nil, err
	}

	mainMetricName := "m_query_time_sum"
	if in.MainMetricName != "" {
		mainMetricName = in.MainMetricName
	}

	return s.rm.SelectFilters(ctx, periodStartFromSec, periodStartToSec, mainMetricName)
}

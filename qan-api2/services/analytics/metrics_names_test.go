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

package analytics

import (
	"context"
	"reflect"
	"testing"

	qanpb "github.com/percona/pmm/api/qanpb"
	"github.com/percona/pmm/qan-api2/models"
)

func TestService_GetMetricsNames(t *testing.T) {
	type fields struct {
		rm models.Reporter
		mm models.Metrics
	}
	type args struct {
		ctx context.Context
		in  *qanpb.MetricsNamesRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *qanpb.MetricsNamesReply
		wantErr bool
	}{
		{
			name:    "success",
			fields:  fields{},
			args:    args{},
			want:    &qanpb.MetricsNamesReply{Data: metricsNames},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				rm: tt.fields.rm,
				mm: tt.fields.mm,
			}
			got, err := s.GetMetricsNames(tt.args.ctx, tt.args.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.GetMetricsNames() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Service.GetMetricsNames() = %v, want %v", got, tt.want)
			}
		})
	}
}

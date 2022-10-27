package telemetry

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	vmAddress = "http://127.0.0.1:9090/prometheus/"
)

type testMetricResult struct {
	key   string
	value string
}

func Test_dataSourceVictoriaMetrics_FetchMetrics(t *testing.T) {
	logger := logrus.StandardLogger()
	logger.SetLevel(logrus.DebugLevel)
	logEntry := logrus.NewEntry(logger)

	type fields struct {
		l      *logrus.Entry
		config DataSourceVictoriaMetrics
		vm     v1.API
	}
	type args struct {
		ctx    context.Context
		config Config
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    func() []testMetricResult
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should return metrics based on the standard config successfully",
			fields: fields{
				config: DataSourceVictoriaMetrics{
					Enabled: true,
					Timeout: 10 * time.Second,
					Address: vmAddress,
				},
			},
			args: args{
				config: Config{
					ID:      "Metric",
					Source:  "VM",
					Query:   "node_uname_info",
					Summary: "should return node process",
					Data: []ConfigData{
						{
							MetricName: "pmm_node_uname_info",
							Label:      "machine",
						},
					},
					DataJson: nil,
				},
			},
			want: func() []testMetricResult {
				resulVector, err := makeTestCallToVM("node_uname_info")
				if err != nil {
					assert.Fail(t, err.Error())
					return nil
				}
				machine := resulVector[0].Metric[model.LabelName("machine")]

				return []testMetricResult{{
					key:   "pmm_node_uname_info",
					value: string(machine),
				}}
			},
			wantErr: assert.NoError,
		},
		{
			name: "should return metrics based on the data json config successfully and data parameter will be ignores in this case",
			fields: fields{
				config: DataSourceVictoriaMetrics{
					Enabled: true,
					Timeout: 10 * time.Second,
					Address: vmAddress,
				},
			},
			args: args{
				config: Config{
					ID:      "Metric",
					Source:  "VM",
					Query:   "node_uname_info",
					Summary: "should return node process",
					Data: []ConfigData{
						{
							MetricName: "pmm_node_uname_info",
							Label:      "machine",
						},
					},
					DataJson: &DataJson{
						MetricName: "pmm_node_uname_info",
						Params: []Param{
							{
								Key:   "arch",
								Label: "machine",
							},
							{
								Key:   "node_type",
								Label: "node_type",
							},
							{
								Key:   "value",
								Value: "1",
							},
						},
					},
				},
			},
			want: func() []testMetricResult {
				resulVector, err := makeTestCallToVM("node_uname_info")
				if err != nil {
					assert.Fail(t, err.Error())
					return nil
				}
				machine := resulVector[0].Metric[model.LabelName("machine")]
				nodeType := resulVector[0].Metric[model.LabelName("node_type")]
				value := resulVector[0].Value

				return []testMetricResult{{
					key:   "pmm_node_uname_info",
					value: fmt.Sprintf(`{"pmm_node_uname_info":[{"arch":"%s","node_type":"%s","value":"%s"}]}`, machine, nodeType, value),
				}}
			},
			wantErr: assert.NoError,
		},
		{
			name: "should return metrics based on the data json config successfully which returns several records in json",
			fields: fields{
				config: DataSourceVictoriaMetrics{
					Enabled: true,
					Timeout: 10 * time.Second,
					Address: vmAddress,
				},
			},
			args: args{
				config: Config{
					ID:      "Metric",
					Source:  "VM",
					Query:   "alertmanager_alerts",
					Summary: "alerts of alertmanager",
					DataJson: &DataJson{
						MetricName: "pmm_alertmanager_alerts",
						Params: []Param{
							{
								Key:   "job",
								Label: "job",
							},
							{
								Key:   "state",
								Label: "state",
							},
						},
					},
				},
			},
			want: func() []testMetricResult {
				resulVector, err := makeTestCallToVM("alertmanager_alerts")
				if err != nil {
					assert.Fail(t, err.Error())
					return nil
				}

				var sb strings.Builder
				for i, m := range resulVector {
					job := m.Metric[model.LabelName("job")]
					state := m.Metric[model.LabelName("state")]

					sb.WriteString(fmt.Sprintf(`{"job":"%s","state":"%s"}`, job, state))
					if i != len(resulVector)-1 {
						sb.WriteString(",")
					}
				}

				return []testMetricResult{{
					key:   "pmm_alertmanager_alerts",
					value: fmt.Sprintf(`{"pmm_alertmanager_alerts":[%s]}`, sb.String()),
				}}
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			d, err := NewDataSourceVictoriaMetrics(tt.fields.config, logEntry)
			if err != nil {
				assert.Fail(t, "cannot initialize victoria metrics source.", err)
				return
			}

			ctx := context.TODO()
			got, err := d.FetchMetrics(ctx, tt.args.config)
			if !tt.wantErr(t, err, fmt.Sprintf("FetchMetrics(%v, %v)", ctx, tt.args.config)) {
				return
			}

			want := tt.want()
			assert.Equal(t, len(want), len(got[0]))
			for i, metricResult := range want {
				assert.Equal(t, metricResult.key, got[0][i].Key)
				assert.Equal(t, metricResult.value, got[0][i].Value)
			}
		})
	}
}

func makeTestCallToVM(query string) (model.Vector, error) {
	client, err := api.NewClient(api.Config{
		Address: vmAddress,
	})
	if err != nil {
		return nil, err
	}

	api := v1.NewAPI(client)
	r, _, err := api.Query(context.Background(), query, time.Now())
	if err != nil {
		return nil, nil
	}

	resulVector := r.(model.Vector)
	return resulVector, nil
}

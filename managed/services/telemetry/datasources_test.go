package telemetry

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func Test_fetchMetricsFromDB(t *testing.T) {
	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	logger := logrus.StandardLogger()
	logger.SetLevel(logrus.DebugLevel)
	logEntry := logrus.NewEntry(logger)

	type args struct {
		timeout time.Duration
		config  Config
	}
	tests := []struct {
		name    string
		args    args
		want    func() []testMetricResult
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should return metric in JSON format successfully",
			args: args{
				timeout: 10 * time.Second,
				config: Config{
					ID:     "Metric ID",
					Source: "PMMDB_SELECT",
					Query:  "agent_type, status from agents",
					DataJson: &DataJson{
						MetricName: "pmm_agents_status_info",
						Params: []Param{
							{
								Key:    "key",
								Column: "agent_type",
							},
							{
								Key:    "value",
								Column: "status",
							},
						},
					},
				},
			},
			want: func() []testMetricResult {
				query, err := sqlDB.Query("select agent_type, status from agents")
				if err != nil {
					assert.Fail(t, err.Error())
					return nil
				}

				var strs []string
				for query.Next() {
					var agentType, status string
					err := query.Scan(&agentType, &status)
					if err != nil {
						assert.Fail(t, err.Error())
						return nil
					}
					strs = append(strs, fmt.Sprintf(`{"key":"%s","value":"%s"}`, agentType, status))
				}
				assert.True(t, len(strs) > 0) //make sure that JSON won't be empty

				return []testMetricResult{{
					key:   "pmm_agents_status_info",
					value: fmt.Sprintf(`{"pmm_agents_status_info":[%s]}`, strings.Join(strs, ",")),
				}}
			},
			wantErr: assert.NoError,
		},
		{
			name: "should return metric in simple format",
			args: args{
				timeout: 10 * time.Second,
				config: Config{
					ID:     "Metric ID",
					Source: "PMMDB_SELECT",
					Query:  "agent_type from agents",
					Data: []ConfigData{
						{
							MetricName: "pmm_agent_type",
							Column:     "agent_type",
						},
					},
				},
			},
			want: func() []testMetricResult {
				query, err := sqlDB.Query("select agent_type from agents")
				if err != nil {
					assert.Fail(t, err.Error())
					return nil
				}

				var res []testMetricResult
				for query.Next() {
					var agentType string
					err := query.Scan(&agentType)
					if err != nil {
						assert.Fail(t, err.Error())
						return nil
					}
					res = append(res, testMetricResult{
						key:   "pmm_agent_type",
						value: agentType,
					})
				}
				assert.True(t, len(res) > 0) //make sure that expected metrics won't be empty
				return res
			},
			wantErr: assert.NoError,
		},
		{
			name: "should return metric in simple format when query with case operator",
			args: args{
				timeout: 10 * time.Second,
				config: Config{
					ID:     "Metric ID",
					Source: "PMMDB_SELECT",
					Query:  "(CASE WHEN ia->'disabled' = 'false' THEN '1' ELSE '0' END) AS ia_enabled FROM settings s, jsonb_extract_path(s.settings, 'alerting') AS ia",
					Data: []ConfigData{
						{
							MetricName: "pmm_server_ia_enabled",
							Column:     "ia_enabled",
						},
					},
				},
			},
			want: func() []testMetricResult {
				return []testMetricResult{
					{
						key:   "pmm_server_ia_enabled",
						value: "1",
					},
				}
			},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			got, err := fetchMetricsFromDB(ctx, logEntry, tt.args.timeout, sqlDB, tt.args.config)
			if !tt.wantErr(t, err, fmt.Sprintf("fetchMetricsFromDB(%v, %v, %v, %v, %v)", ctx, logEntry, tt.args.timeout, sqlDB, tt.args.config)) {
				return
			}
			want := tt.want()
			assert.Equal(t, len(want), len(got))
			for i, metricResult := range want {
				assert.Equal(t, metricResult.key, got[i][0].Key)
				assert.Equal(t, metricResult.value, got[i][0].Value)
			}
		})
	}
}

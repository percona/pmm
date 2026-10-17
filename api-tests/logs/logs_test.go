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

package logs

import (
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	logsClient "github.com/percona/pmm/api/logs/v1/json/client"
	"github.com/percona/pmm/api/logs/v1/json/client/logs_service"
)

// lastHour returns the start and end of the last hour; PMM Server's own logs always have lines in it.
func lastHour() (strfmt.DateTime, strfmt.DateTime) {
	end := time.Now().UTC()
	return strfmt.DateTime(end.Add(-time.Hour)), strfmt.DateTime(end)
}

func search(t *testing.T, cursor string, limit int32) *logs_service.SearchLogsOKBody {
	t.Helper()
	start, end := lastHour()
	res, err := logsClient.Default.LogsService.SearchLogs(&logs_service.SearchLogsParams{
		Body: logs_service.SearchLogsBody{
			Filter: &logs_service.SearchLogsParamsBodyFilter{Start: start, End: end},
			Cursor: cursor,
			Limit:  limit,
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	return res.Payload
}

func TestSearchLogs(t *testing.T) {
	t.Parallel()

	first := search(t, "", 50)
	require.Len(t, first.Lines, 50)
	require.NotEmpty(t, first.NextCursor)
	for _, l := range first.Lines {
		assert.NotEmpty(t, l.Source)
		assert.NotEmpty(t, l.NodeName)
	}

	second := search(t, first.NextCursor, 50)
	require.NotEmpty(t, second.Lines)
	key := func(l *logs_service.SearchLogsOKBodyLinesItems0) string {
		return l.Time.String() + "|" + l.Source + "|" + l.FileName + "|" + l.Body
	}
	seen := make(map[string]bool, len(first.Lines))
	for _, l := range first.Lines {
		seen[key(l)] = true
	}
	for _, l := range second.Lines {
		assert.False(t, seen[key(l)], "line returned twice: %s", key(l))
		assert.False(t, time.Time(l.Time).After(time.Time(first.Lines[len(first.Lines)-1].Time)))
	}
}

func TestGetLogVolume(t *testing.T) {
	t.Parallel()

	start, end := lastHour()
	res, err := logsClient.Default.LogsService.GetLogVolume(&logs_service.GetLogVolumeParams{
		Body:    logs_service.GetLogVolumeBody{Filter: &logs_service.GetLogVolumeParamsBodyFilter{Start: start, End: end}},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	assert.Equal(t, "60s", res.Payload.Step)
	assert.NotEmpty(t, res.Payload.Buckets)
	assert.LessOrEqual(t, len(res.Payload.Buckets), 100)
}

func TestGetLogPatterns(t *testing.T) {
	t.Parallel()

	start, end := lastHour()
	res, err := logsClient.Default.LogsService.GetLogPatterns(&logs_service.GetLogPatternsParams{
		Body:    logs_service.GetLogPatternsBody{Filter: &logs_service.GetLogPatternsParamsBodyFilter{Start: start, End: end}},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	patterns := res.Payload.Patterns
	require.NotEmpty(t, patterns)
	assert.LessOrEqual(t, len(patterns), 50)
	prev := int64(-1)
	for i, p := range patterns {
		count, err := strconv.ParseInt(p.Count, 10, 64)
		require.NoError(t, err)
		if i > 0 {
			assert.LessOrEqual(t, count, prev, "patterns are not sorted by count")
		}
		prev = count
		assert.NotEmpty(t, p.Sample)
	}
}

func TestGetLogContext(t *testing.T) {
	t.Parallel()

	line := search(t, "", 1).Lines[0]
	res, err := logsClient.Default.LogsService.GetLogContext(&logs_service.GetLogContextParams{
		Body: logs_service.GetLogContextBody{
			Time:     line.Time,
			NodeName: line.NodeName,
			FileName: line.FileName,
			Lines:    5,
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	lines := res.Payload.Lines
	require.NotEmpty(t, lines)
	assert.LessOrEqual(t, len(lines), 10)
	require.Less(t, int(res.Payload.AnchorIndex), len(lines))
	anchor := lines[res.Payload.AnchorIndex]
	assert.Equal(t, line.FileName, anchor.FileName)
	assert.False(t, time.Time(anchor.Time).After(time.Time(line.Time)))
}

func TestGetLogFacets(t *testing.T) {
	t.Parallel()

	start, end := lastHour()
	res, err := logsClient.Default.LogsService.GetLogFacets(&logs_service.GetLogFacetsParams{
		Body:    logs_service.GetLogFacetsBody{Filter: &logs_service.GetLogFacetsParamsBodyFilter{Start: start, End: end}},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	sources := make([]string, 0, len(res.Payload.Sources))
	for _, s := range res.Payload.Sources {
		sources = append(sources, s.Value)
	}
	assert.Contains(t, sources, "pmm-managed")
	assert.NotEmpty(t, res.Payload.NodeNames)
	assert.NotEmpty(t, res.Payload.Presets)
}

func TestBadRange(t *testing.T) {
	t.Parallel()

	start, end := lastHour()
	_, err := logsClient.Default.LogsService.SearchLogs(&logs_service.SearchLogsParams{
		Body:    logs_service.SearchLogsBody{Filter: &logs_service.SearchLogsParamsBodyFilter{Start: end, End: start}},
		Context: pmmapitests.Context,
	})
	pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "The end of the time range is before its start.")
}

func TestViewerCanRead(t *testing.T) {
	t.Parallel()

	gClient := pmmapitests.GetGrafanaClient(t)
	login := pmmapitests.TestString(t, "logs-viewer")
	password := pmmapitests.TestString(t, "logs-viewer-password")
	created, err := gClient.AdminUsers.AdminCreateUser(&models.AdminCreateUserForm{
		Name:     login,
		Login:    login,
		Password: models.Password(password),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = gClient.AdminUsers.AdminDeleteUser(created.Payload.ID)
	})
	_, err = gClient.Org.UpdateOrgUserForCurrentOrg(created.Payload.ID, &models.UpdateOrgUserCommand{Role: "Viewer"})
	require.NoError(t, err)

	viewerURL := *pmmapitests.BaseURL
	viewerURL.User = url.UserPassword(login, password)
	viewer := logsClient.New(pmmapitests.Transport(&viewerURL, pmmapitests.ServerInsecureTLS), nil)
	start, end := lastHour()
	res, err := viewer.LogsService.GetLogVolume(&logs_service.GetLogVolumeParams{
		Body:    logs_service.GetLogVolumeBody{Filter: &logs_service.GetLogVolumeParamsBodyFilter{Start: start, End: end}},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, res.Payload.Buckets)
}

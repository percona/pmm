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
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	accesscontrolClient "github.com/percona/pmm/api/accesscontrol/v1beta1/json/client"
	"github.com/percona/pmm/api/accesscontrol/v1beta1/json/client/access_control_service"
	logsClient "github.com/percona/pmm/api/logs/v1/json/client"
	"github.com/percona/pmm/api/logs/v1/json/client/logs_service"
	serverClient "github.com/percona/pmm/api/server/v1/json/client"
	server "github.com/percona/pmm/api/server/v1/json/client/server_service"
)

// lastHour returns the start and end of the last hour; PMM Server's own logs always have lines in it.
func lastHour() (strfmt.DateTime, strfmt.DateTime) {
	end := time.Now().UTC()
	return strfmt.DateTime(end.Add(-time.Hour)), strfmt.DateTime(end)
}

func search(t *testing.T, start, end strfmt.DateTime, cursor string, limit int32) *logs_service.SearchLogsOKBody {
	t.Helper()
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

	// A window that ended a while ago, so no new lines arrive in it between the calls.
	now := time.Now().UTC()
	start, end := strfmt.DateTime(now.Add(-62*time.Minute)), strfmt.DateTime(now.Add(-2*time.Minute))
	first := search(t, start, end, "", 50)
	require.Len(t, first.Lines, 50)
	require.NotEmpty(t, first.NextCursor)
	for _, l := range first.Lines {
		assert.NotEmpty(t, l.Source)
		assert.NotEmpty(t, l.NodeName)
	}
	second := search(t, start, end, first.NextCursor, 50)
	require.NotEmpty(t, second.Lines)
	assert.False(t, time.Time(second.Lines[0].Time).After(time.Time(first.Lines[len(first.Lines)-1].Time)))

	// Identical lines are legitimate, so two pages must hold the same lines, with the same counts, as one page of 100.
	key := func(l *logs_service.SearchLogsOKBodyLinesItems0) string {
		return fmt.Sprint(l.Time.String(), l.Source, l.FileName, l.Body, l.Attributes)
	}
	paged := make(map[string]int)
	for _, l := range append(first.Lines, second.Lines...) {
		paged[key(l)]++
	}
	whole := make(map[string]int)
	for _, l := range search(t, start, end, "", int32(len(first.Lines)+len(second.Lines))).Lines {
		whole[key(l)]++
	}
	assert.Equal(t, whole, paged)
}

func TestSearchLogs_SeverityRangeAndExcludedNodes(t *testing.T) {
	t.Parallel()

	start, end := lastHour()
	searchFiltered := func(f *logs_service.SearchLogsParamsBodyFilter) []*logs_service.SearchLogsOKBodyLinesItems0 {
		f.Start, f.End = start, end
		res, err := logsClient.Default.LogsService.SearchLogs(&logs_service.SearchLogsParams{
			Body:    logs_service.SearchLogsBody{Filter: f, Limit: 100},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		return res.Payload.Lines
	}

	info := searchFiltered(&logs_service.SearchLogsParamsBodyFilter{
		NodeNames: []string{"pmm-server"}, MinSeverity: 9, MaxSeverity: 12,
	})
	require.NotEmpty(t, info)
	for _, l := range info {
		assert.GreaterOrEqual(t, l.SeverityNumber, int32(9))
		assert.LessOrEqual(t, l.SeverityNumber, int32(12))
	}

	excluded := searchFiltered(&logs_service.SearchLogsParamsBodyFilter{
		NodeNames: []string{"pmm-server"}, ExcludeNodeNames: []string{"pmm-server"},
	})
	assert.Empty(t, excluded)
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

	// The swagger client sends times in milliseconds (strfmt.MarshalFormat); pmm-managed's log has millisecond times.
	start, end := lastHour()
	found, err := logsClient.Default.LogsService.SearchLogs(&logs_service.SearchLogsParams{
		Body: logs_service.SearchLogsBody{
			Filter: &logs_service.SearchLogsParamsBodyFilter{Start: start, End: end, Sources: []string{"pmm-managed"}},
			Limit:  1,
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	require.NotEmpty(t, found.Payload.Lines)
	line := found.Payload.Lines[0]
	res, err := logsClient.Default.LogsService.GetLogContext(&logs_service.GetLogContextParams{
		Body: logs_service.GetLogContextBody{
			Time:     line.Time,
			NodeName: line.NodeName,
			FileName: line.FileName,
			Source:   line.Source,
			Lines:    5,
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	lines := res.Payload.Lines
	require.NotEmpty(t, lines)
	require.Less(t, int(res.Payload.AnchorIndex), len(lines))
	anchor := lines[res.Payload.AnchorIndex]
	assert.Equal(t, line.FileName, anchor.FileName)
	assert.True(t, time.Time(anchor.Time).Equal(time.Time(line.Time)))
	inContext := false
	for _, l := range lines {
		inContext = inContext || (time.Time(l.Time).Equal(time.Time(line.Time)) && strings.HasPrefix(l.Body, line.Body))
	}
	assert.True(t, inContext, "the line itself is in its context")
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

	_, viewer := createViewer(t, "logs-viewer")
	start, end := lastHour()
	res, err := viewer.LogsService.GetLogVolume(&logs_service.GetLogVolumeParams{
		Body:    logs_service.GetLogVolumeBody{Filter: &logs_service.GetLogVolumeParamsBodyFilter{Start: start, End: end}},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, res.Payload.Buckets)
}

// createViewer creates a Grafana user with the Viewer role and returns its ID and a logs client acting as it.
func createViewer(t *testing.T, name string) (int64, *logsClient.PMMLogsAPI) {
	t.Helper()
	gClient := pmmapitests.GetGrafanaClient(t)
	login := pmmapitests.TestString(t, name)
	password := pmmapitests.TestString(t, name+"-password")
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

	u := *pmmapitests.BaseURL
	u.User = url.UserPassword(login, password)
	return created.Payload.ID, logsClient.New(pmmapitests.Transport(&u, pmmapitests.ServerInsecureTLS), nil)
}

// TestLBACDenied changes the server-wide access control setting, so it does not run in parallel.
func TestLBACDenied(t *testing.T) { //nolint:paralleltest
	settings, err := serverClient.Default.ServerService.GetSettings(&server.GetSettingsParams{Context: pmmapitests.Context})
	require.NoError(t, err)
	wasEnabled := settings.Payload.Settings.EnableAccessControl
	setAccessControl := func(enabled bool) {
		_, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
			Body:    server.ChangeSettingsBody{EnableAccessControl: new(enabled)},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
	}
	setAccessControl(true)
	t.Cleanup(func() { setAccessControl(wasEnabled) })

	accessControl := accesscontrolClient.New(pmmapitests.Transport(pmmapitests.BaseURL, pmmapitests.ServerInsecureTLS), nil)
	role, err := accessControl.AccessControlService.CreateRole(&access_control_service.CreateRoleParams{
		Body: access_control_service.CreateRoleBody{
			Title:  pmmapitests.TestString(t, "logs-lbac"),
			Filter: `{service_name="logs-lbac-test"}`,
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = accessControl.AccessControlService.DeleteRole(&access_control_service.DeleteRoleParams{
			RoleID:  role.Payload.RoleID,
			Context: pmmapitests.Context,
		})
	})

	userID, viewer := createViewer(t, "logs-lbac-viewer")
	_, err = accessControl.AccessControlService.AssignRoles(&access_control_service.AssignRolesParams{
		Body:    access_control_service.AssignRolesBody{RoleIds: []int64{role.Payload.RoleID}, UserID: userID},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)

	start, end := lastHour()
	// The auth server caches the access control setting for a few seconds.
	require.Eventually(t, func() bool {
		_, err = viewer.LogsService.GetLogVolume(&logs_service.GetLogVolumeParams{
			Body:    logs_service.GetLogVolumeBody{Filter: &logs_service.GetLogVolumeParamsBodyFilter{Start: start, End: end}},
			Context: pmmapitests.Context,
		})
		return err != nil
	}, 15*time.Second, time.Second)
	pmmapitests.AssertAPIErrorf(t, err, 403, codes.PermissionDenied, "Logs are not available to users with label-based access filters.")

	_, err = viewer.LogsService.SearchLogs(&logs_service.SearchLogsParams{
		Body:    logs_service.SearchLogsBody{Filter: &logs_service.SearchLogsParamsBodyFilter{Start: start, End: end}},
		Context: pmmapitests.Context,
	})
	pmmapitests.AssertAPIErrorf(t, err, 403, codes.PermissionDenied, "Logs are not available to users with label-based access filters.")
}

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

package server

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	advisorClient "github.com/percona/pmm/api/advisors/v1/json/client"
	advisor "github.com/percona/pmm/api/advisors/v1/json/client/advisor_service"
	managementClient "github.com/percona/pmm/api/management/v1/json/client"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
	serverClient "github.com/percona/pmm/api/server/v1/json/client"
	"github.com/percona/pmm/api/server/v1/json/client/server_service"
)

// customCheckScript reports three warnings, so every run of the check yields three insights.
const customCheckScript = `def check_context(docs, context):
    print("rows:", len(docs[0]))
    return [{"summary": "Finding " + str(i), "description": "Found by an API test.", "severity": "warning"} for i in range(3)]
`

func TestStartChecks(t *testing.T) {
	t.Run("with advisors enabled", func(t *testing.T) {
		toggleAdvisorChecks(t, true)
		t.Cleanup(func() { RestoreSettingsDefaults(t) })
		waitForNoActiveRun(t)

		resp, err := advisorClient.Default.AdvisorService.StartAdvisorChecks(nil)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotEmpty(t, resp.Payload.RunID)
	})

	t.Run("only one run at a time", func(t *testing.T) {
		toggleAdvisorChecks(t, true)
		t.Cleanup(func() { RestoreSettingsDefaults(t) })
		waitForNoActiveRun(t)

		// Concurrent requests race for the single run slot. A run can finish
		// before the last request arrives, so more than one may start, but each
		// request either starts a run or is rejected, and the runs never overlap.
		const requests = 5
		runIDs := make([]string, requests)
		errs := make([]error, requests)
		var wg sync.WaitGroup
		for i := range requests {
			wg.Go(func() {
				resp, err := advisorClient.Default.AdvisorService.StartAdvisorChecks(nil)
				errs[i] = err
				if err == nil {
					runIDs[i] = resp.Payload.RunID
				}
			})
		}
		wg.Wait()

		started := make(map[string]struct{}, requests)
		for i, err := range errs {
			if err != nil {
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, "Advisor checks are already running")
				continue
			}
			started[runIDs[i]] = struct{}{}
		}
		require.NotEmpty(t, started)

		waitForNoActiveRun(t)
		resp, err := advisorClient.Default.AdvisorService.ListRuns(&advisor.ListRunsParams{
			PageSize: new(int32(20)),
			Context:  pmmapitests.Context,
		})
		require.NoError(t, err)

		var runs []*advisor.ListRunsOKBodyResultsItems0
		for _, run := range resp.Payload.Results {
			if _, ok := started[run.ID]; ok {
				runs = append(runs, run)
			}
		}
		require.Len(t, runs, len(started))
		// newest first, so each run must start after the next one in the list finished
		for i := 1; i < len(runs); i++ {
			newer, older := runs[i-1], runs[i]
			assert.False(t, time.Time(newer.StartedAt).Before(time.Time(older.FinishedAt)),
				"run %s started before run %s finished", newer.ID, older.ID)
		}
	})

	t.Run("a run keeps its scope and leaves nothing pending", func(t *testing.T) {
		toggleAdvisorChecks(t, true)
		t.Cleanup(func() { RestoreSettingsDefaults(t) })
		waitForNoActiveRun(t)

		checks, err := advisorClient.Default.AdvisorService.ListAdvisorChecks(nil)
		require.NoError(t, err)
		require.NotEmpty(t, checks.Payload.Checks)
		name := checks.Payload.Checks[0].Name

		resp, err := advisorClient.Default.AdvisorService.StartAdvisorChecks(&advisor.StartAdvisorChecksParams{
			Body:    advisor.StartAdvisorChecksBody{Names: []string{name}},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		runID := resp.Payload.RunID

		waitForNoActiveRun(t)
		runs, err := advisorClient.Default.AdvisorService.ListRuns(&advisor.ListRunsParams{
			RunID:   &runID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		require.Len(t, runs.Payload.Results, 1)
		run := runs.Payload.Results[0]
		assert.Equal(t, runID, run.ID)
		assert.Equal(t, []string{name}, run.CheckNames)
		assert.LessOrEqual(t, run.ChecksCount, run.PlannedChecksCount)
		assert.LessOrEqual(t, run.ServicesCount, run.PlannedServicesCount)

		pending := "ADVISOR_CHECK_RESULT_STATUS_PENDING"
		insights, err := advisorClient.Default.AdvisorService.ListInsights(&advisor.ListInsightsParams{
			RunID:   &runID,
			Status:  &pending,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Empty(t, insights.Payload.Results)
	})

	t.Run("with advisors disabled", func(t *testing.T) {
		toggleAdvisorChecks(t, false)
		t.Cleanup(func() { RestoreSettingsDefaults(t) })

		resp, err := advisorClient.Default.AdvisorService.StartAdvisorChecks(nil)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, `advisor checks are disabled.`)
		assert.Nil(t, resp)
	})
}

func TestListAdvisorChecks(t *testing.T) {
	toggleAdvisorChecks(t, true)
	t.Cleanup(func() { RestoreSettingsDefaults(t) })

	resp, err := advisorClient.Default.AdvisorService.ListAdvisorChecks(nil)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Payload.Checks)
	for _, c := range resp.Payload.Checks {
		assert.NotEmpty(t, c.Name, "%+v", c)
		assert.NotEmpty(t, c.Summary, "%+v", c)
		assert.NotEmpty(t, c.Description, "%+v", c)
		assert.NotEmpty(t, c.Category, "%+v", c)
	}
	assert.True(t, slices.IsSortedFunc(resp.Payload.Checks, func(a, b *advisor.ListAdvisorChecksOKBodyChecksItems0) int {
		return strings.Compare(a.Name, b.Name)
	}), "checks must be ordered by name")
}

func TestChangeAdvisorChecks(t *testing.T) {
	toggleAdvisorChecks(t, true)
	t.Cleanup(func() { RestoreSettingsDefaults(t) })

	t.Run("enable disable", func(t *testing.T) {
		t.Run("enable disable", func(t *testing.T) {
			resp, err := advisorClient.Default.AdvisorService.ListAdvisorChecks(nil)
			require.NoError(t, err)
			require.NotEmpty(t, resp.Payload.Checks)

			var check *advisor.ListAdvisorChecksOKBodyChecksItems0

			// enable ⥁ disable loop, it checks current state of first returned check and changes its state,
			// then in second iteration it returns state to its origin.
			for range 2 {
				check = resp.Payload.Checks[0]
				params := &advisor.ChangeAdvisorChecksParams{
					Body: advisor.ChangeAdvisorChecksBody{
						Params: []*advisor.ChangeAdvisorChecksParamsBodyParamsItems0{
							{
								Name:   check.Name,
								Enable: new(!check.Enabled),
							},
						},
					},
					Context: pmmapitests.Context,
				}

				_, err = advisorClient.Default.AdvisorService.ChangeAdvisorChecks(params)
				require.NoError(t, err)

				resp, err = advisorClient.Default.AdvisorService.ListAdvisorChecks(nil)
				require.NoError(t, err)
				require.NotEmpty(t, resp.Payload.Checks)

				for _, c := range resp.Payload.Checks {
					if c.Name == check.Name {
						assert.NotEqual(t, check.Enabled, c.Enabled)
						break
					}
				}
			}
		})
	})

	t.Run("per service", func(t *testing.T) {
		serviceID, _ := addAdvisorTargetService(t)
		name := customCheckName(t)
		createCustomCheck(t, name)

		change := func(p *advisor.ChangeAdvisorChecksParamsBodyParamsItems0) error {
			_, err := advisorClient.Default.AdvisorService.ChangeAdvisorChecks(&advisor.ChangeAdvisorChecksParams{
				Body:    advisor.ChangeAdvisorChecksBody{Params: []*advisor.ChangeAdvisorChecksParamsBodyParamsItems0{p}},
				Context: pmmapitests.Context,
			})
			return err
		}
		listed := func(t *testing.T) *advisor.ListAdvisorChecksOKBodyChecksItems0 {
			t.Helper()
			resp, err := advisorClient.Default.AdvisorService.ListAdvisorChecks(nil)
			require.NoError(t, err)
			for _, c := range resp.Payload.Checks {
				if c.Name == name {
					return c
				}
			}
			require.FailNow(t, "check isn't listed", name)
			return nil
		}

		t.Run("disable and enable", func(t *testing.T) {
			err := change(&advisor.ChangeAdvisorChecksParamsBodyParamsItems0{Name: name, ServiceIds: []string{serviceID}, Enable: new(false)})
			require.NoError(t, err)
			c := listed(t)
			assert.True(t, c.Enabled, "a per-service disable must leave the check enabled globally")
			assert.Equal(t, []string{serviceID}, c.DisabledServiceIds)

			err = change(&advisor.ChangeAdvisorChecksParamsBodyParamsItems0{Name: name, ServiceIds: []string{serviceID}, Enable: new(true)})
			require.NoError(t, err)
			c = listed(t)
			assert.True(t, c.Enabled)
			assert.Empty(t, c.DisabledServiceIds)
		})

		t.Run("enable flag is required", func(t *testing.T) {
			err := change(&advisor.ChangeAdvisorChecksParamsBodyParamsItems0{Name: name, ServiceIds: []string{serviceID}})
			pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Enable flag is required to change check %s per service.", name)
		})

		t.Run("unknown service", func(t *testing.T) {
			err := change(&advisor.ChangeAdvisorChecksParamsBodyParamsItems0{Name: name, ServiceIds: []string{"unknown-service-id"}, Enable: new(false)})
			pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "service with ID unknown-service-id not found")
			assert.Empty(t, listed(t).DisabledServiceIds)
		})

		t.Run("check disabled globally", func(t *testing.T) {
			err := change(&advisor.ChangeAdvisorChecksParamsBodyParamsItems0{Name: name, Enable: new(false)})
			require.NoError(t, err)
			t.Cleanup(func() {
				assert.NoError(t, change(&advisor.ChangeAdvisorChecksParamsBodyParamsItems0{Name: name, Enable: new(true)}))
			})

			err = change(&advisor.ChangeAdvisorChecksParamsBodyParamsItems0{Name: name, ServiceIds: []string{serviceID}, Enable: new(false)})
			pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition,
				"advisor check %s is disabled globally; enable it globally to manage per-service settings", name)
		})
	})
}

func TestCustomAdvisorCheck(t *testing.T) {
	toggleAdvisorChecks(t, true)
	t.Cleanup(func() { RestoreSettingsDefaults(t) })

	serviceID, serviceName := addAdvisorTargetService(t)
	name := customCheckName(t)

	created := createCustomCheck(t, name)
	assert.Equal(t, name, created.Name)
	assert.True(t, created.UserDefined)
	assert.True(t, created.Enabled)

	t.Run("get", func(t *testing.T) {
		resp, err := advisorClient.Default.AdvisorService.GetAdvisorCheck(&advisor.GetAdvisorCheckParams{
			Name:    name,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		c := resp.Payload.Check
		assert.Equal(t, name, c.Name)
		assert.Equal(t, "API test check", c.Summary)
		assert.Equal(t, "API tests", c.Category)
		assert.Equal(t, advisor.GetAdvisorCheckOKBodyCheckTechnologyADVISORCHECKTECHNOLOGYMYSQL, *c.Technology)
		assert.Equal(t, customCheckScript, c.Script)
		require.Len(t, c.Queries, 1)
		assert.Equal(t, "METRICS_INSTANT", c.Queries[0].Type)
		assert.Equal(t, "vector(1)", c.Queries[0].Query)
		assert.True(t, c.UserDefined)
		assert.True(t, c.Enabled)
	})

	t.Run("listed", func(t *testing.T) {
		resp, err := advisorClient.Default.AdvisorService.ListAdvisorChecks(nil)
		require.NoError(t, err)
		var found *advisor.ListAdvisorChecksOKBodyChecksItems0
		for _, c := range resp.Payload.Checks {
			if c.Name == name {
				found = c
				break
			}
		}
		require.NotNil(t, found, "check %s is not listed", name)
		assert.True(t, found.UserDefined)
		assert.Equal(t, "API tests", found.Category)
	})

	t.Run("update", func(t *testing.T) {
		body := updateCheckBody()
		body.Summary = "Updated API test check"
		_, err := advisorClient.Default.AdvisorService.UpdateAdvisorCheck(&advisor.UpdateAdvisorCheckParams{
			Name:    name,
			Body:    advisor.UpdateAdvisorCheckBody{Check: body},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		resp, err := advisorClient.Default.AdvisorService.GetAdvisorCheck(&advisor.GetAdvisorCheckParams{
			Name:    name,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Equal(t, "Updated API test check", resp.Payload.Check.Summary)
		assert.True(t, resp.Payload.Check.UserDefined)
	})

	t.Run("test against a target", func(t *testing.T) {
		targets, err := advisorClient.Default.AdvisorService.ListAdvisorCheckTestTargets(&advisor.ListAdvisorCheckTestTargetsParams{
			Technology: new("ADVISOR_CHECK_TECHNOLOGY_MYSQL"),
			Context:    pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Contains(t, targets.Payload.Targets, &advisor.ListAdvisorCheckTestTargetsOKBodyTargetsItems0{
			ServiceID:   serviceID,
			ServiceName: serviceName,
		})

		resp, err := advisorClient.Default.AdvisorService.TestAdvisorCheck(&advisor.TestAdvisorCheckParams{
			Body: advisor.TestAdvisorCheckBody{
				ServiceID: serviceID,
				Check:     testCheckBody(name, customCheckScript),
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		require.Len(t, resp.Payload.Results, 3)
		for i, r := range resp.Payload.Results {
			assert.Equal(t, fmt.Sprintf("Finding %d", i), r.Summary)
			assert.Equal(t, advisor.TestAdvisorCheckOKBodyResultsItems0SeveritySEVERITYWARNING, *r.Severity)
			assert.Equal(t, name, r.CheckName)
			assert.Equal(t, serviceID, r.ServiceID)
			assert.Equal(t, serviceName, r.ServiceName)
		}
		assert.Equal(t, name+" -> check_context:2:10 -> rows: 1", resp.Payload.ScriptOutput)
	})

	t.Run("test with a failing script", func(t *testing.T) {
		script := "def check_context(docs, context):\n    fail(\"broken check\")\n"
		resp, err := advisorClient.Default.AdvisorService.TestAdvisorCheck(&advisor.TestAdvisorCheckParams{
			Body: advisor.TestAdvisorCheckBody{
				ServiceID: serviceID,
				Check:     testCheckBody(name, script),
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition,
			"Failed to execute check %s on service '%s': ", name, serviceName)
		require.ErrorContains(t, err, "broken check")
		assert.Nil(t, resp)
	})

	t.Run("invalid creates are rejected", func(t *testing.T) {
		noQueries := customCheckBody(customCheckName(t))
		noQueries.Queries = nil
		tabs := customCheckBody(customCheckName(t))
		tabs.Script = "def check_context(docs, context):\n\treturn []\n"

		for _, tc := range []struct {
			name     string
			check    *advisor.CreateAdvisorCheckParamsBodyCheck
			code     int
			grpcCode codes.Code
			message  string
		}{
			{"duplicate name", customCheckBody(name), 409, codes.AlreadyExists, "advisor check " + name + " already exists"},
			{"no user prefix", customCheckBody("api_test_check"), 400, codes.InvalidArgument, "user check name must start with custom_"},
			{"no queries", noQueries, 400, codes.InvalidArgument, "invalid advisor check: check should have at least one query"},
			{"tab indentation", tabs, 400, codes.InvalidArgument, "invalid advisor check: script should use spaces for indentation, not tabs"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				resp, err := advisorClient.Default.AdvisorService.CreateAdvisorCheck(&advisor.CreateAdvisorCheckParams{
					Body:    advisor.CreateAdvisorCheckBody{Check: tc.check},
					Context: pmmapitests.Context,
				})
				pmmapitests.AssertAPIErrorf(t, err, tc.code, tc.grpcCode, tc.message)
				assert.Nil(t, resp)
			})
		}
	})

	t.Run("unknown check", func(t *testing.T) {
		unknown := customCheckName(t)

		updated, err := advisorClient.Default.AdvisorService.UpdateAdvisorCheck(&advisor.UpdateAdvisorCheckParams{
			Name:    unknown,
			Body:    advisor.UpdateAdvisorCheckBody{Check: updateCheckBody()},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Advisor check %s not found", unknown)
		assert.Nil(t, updated)

		deleted, err := advisorClient.Default.AdvisorService.DeleteAdvisorCheck(&advisor.DeleteAdvisorCheckParams{
			Name:    unknown,
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Advisor check %s not found", unknown)
		assert.Nil(t, deleted)
	})

	t.Run("Percona checks cannot be deleted", func(t *testing.T) {
		checks, err := advisorClient.Default.AdvisorService.ListAdvisorChecks(nil)
		require.NoError(t, err)
		var builtin string
		for _, c := range checks.Payload.Checks {
			if !c.UserDefined {
				builtin = c.Name
				break
			}
		}
		require.NotEmpty(t, builtin, "no Percona-shipped check is loaded")

		resp, err := advisorClient.Default.AdvisorService.DeleteAdvisorCheck(&advisor.DeleteAdvisorCheckParams{
			Name:    builtin,
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition,
			"Advisor check %s is shipped by Percona and cannot be modified", builtin)
		assert.Nil(t, resp)
	})

	t.Run("delete", func(t *testing.T) {
		_, err := advisorClient.Default.AdvisorService.DeleteAdvisorCheck(&advisor.DeleteAdvisorCheckParams{
			Name:    name,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)

		resp, err := advisorClient.Default.AdvisorService.GetAdvisorCheck(&advisor.GetAdvisorCheckParams{
			Name:    name,
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Advisor check %s not found", name)
		assert.Nil(t, resp)
	})
}

func TestAdvisorRunsAndInsights(t *testing.T) {
	toggleAdvisorChecks(t, true)
	t.Cleanup(func() { RestoreSettingsDefaults(t) })

	serviceID, serviceName := addAdvisorTargetService(t)
	checkName := customCheckName(t)
	createCustomCheck(t, checkName)

	firstRunID := startAdvisorRun(t, checkName, serviceID)
	runID := startAdvisorRun(t, checkName, serviceID)

	t.Run("run totals", func(t *testing.T) {
		resp, err := advisorClient.Default.AdvisorService.ListRuns(&advisor.ListRunsParams{
			RunID:   &runID,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		require.Len(t, resp.Payload.Results, 1)
		run := resp.Payload.Results[0]
		assert.Equal(t, advisor.ListRunsOKBodyResultsItems0StatusADVISORRUNSTATUSCOMPLETED, *run.Status)
		assert.Equal(t, advisor.ListRunsOKBodyResultsItems0TriggeredByADVISORCHECKTRIGGEREDBYUSER, *run.TriggeredBy)
		assert.Equal(t, []string{checkName}, run.CheckNames)
		assert.Equal(t, []string{serviceID}, run.ServiceIds)
		assert.Equal(t, int32(1), run.PlannedChecksCount)
		assert.Equal(t, int32(1), run.PlannedServicesCount)
		assert.Equal(t, int32(1), run.ChecksCount)
		assert.Equal(t, int32(1), run.ServicesCount)
		assert.Equal(t, int32(3), run.FindingsCount)
		assert.Zero(t, run.ErrorsCount)
		require.Len(t, run.SeverityCounts, 1)
		assert.Equal(t, advisor.ListRunsOKBodyResultsItems0SeverityCountsItems0SeveritySEVERITYWARNING, *run.SeverityCounts[0].Severity)
		assert.Equal(t, int32(3), run.SeverityCounts[0].Count)
	})

	t.Run("insights", func(t *testing.T) {
		var insights []*advisor.ListInsightsOKBodyResultsItems0
		for page, size := range []int{2, 1} {
			resp, err := advisorClient.Default.AdvisorService.ListInsights(&advisor.ListInsightsParams{
				RunID:     &runID,
				PageSize:  new(int32(2)),
				PageIndex: new(int32(page)),
				Context:   pmmapitests.Context,
			})
			require.NoError(t, err)
			assert.Equal(t, int32(3), resp.Payload.TotalItems)
			assert.Equal(t, int32(2), resp.Payload.TotalPages)
			require.Len(t, resp.Payload.Results, size)
			insights = append(insights, resp.Payload.Results...)
		}

		ids := make(map[string]struct{}, len(insights))
		summaries := make([]string, 0, len(insights))
		for _, insight := range insights {
			ids[insight.ID] = struct{}{}
			summaries = append(summaries, insight.Summary)
			assert.Equal(t, runID, insight.RunID)
			assert.Equal(t, checkName, insight.CheckName)
			assert.Equal(t, "API tests", insight.Category)
			assert.Equal(t, serviceID, insight.ServiceID)
			assert.Equal(t, serviceName, insight.ServiceName)
			assert.Equal(t, "mysql", insight.ServiceType)
			assert.Equal(t, "pmm-server", insight.NodeID)
			assert.Equal(t, "pmm-server", insight.NodeName)
			assert.Equal(t, advisor.ListInsightsOKBodyResultsItems0StatusADVISORCHECKRESULTSTATUSFAILED, *insight.Status)
			assert.Equal(t, advisor.ListInsightsOKBodyResultsItems0SeveritySEVERITYWARNING, *insight.Severity)
			assert.Equal(t, advisor.ListInsightsOKBodyResultsItems0TriggeredByADVISORCHECKTRIGGEREDBYUSER, *insight.TriggeredBy)
			assert.Equal(t, "Found by an API test.", insight.Outcome)
			assert.False(t, time.Time(insight.CheckedAt).IsZero())
			assert.False(t, insight.IsRead)
		}
		assert.Len(t, ids, 3, "pages overlap")
		assert.ElementsMatch(t, []string{"Finding 0", "Finding 1", "Finding 2"}, summaries)
	})

	t.Run("filter values", func(t *testing.T) {
		resp, err := advisorClient.Default.AdvisorService.ListInsightsFilterValues(nil)
		require.NoError(t, err)
		assert.Contains(t, resp.Payload.ServiceNames, serviceName)
		assert.Contains(t, resp.Payload.NodeNames, "pmm-server")
	})

	t.Run("mark read", func(t *testing.T) {
		listIDs := func(isRead bool) []string {
			resp, err := advisorClient.Default.AdvisorService.ListInsights(&advisor.ListInsightsParams{
				RunID:   &runID,
				IsRead:  &isRead,
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)
			ids := make([]string, 0, len(resp.Payload.Results))
			for _, insight := range resp.Payload.Results {
				assert.Equal(t, isRead, insight.IsRead)
				ids = append(ids, insight.ID)
			}
			return ids
		}
		markRead := func(body advisor.MarkInsightsReadBody) {
			_, err := advisorClient.Default.AdvisorService.MarkInsightsRead(&advisor.MarkInsightsReadParams{
				Body:    body,
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)
		}

		unread := listIDs(false)
		require.Len(t, unread, 3)
		require.Empty(t, listIDs(true))

		markRead(advisor.MarkInsightsReadBody{Ids: unread[:1], IsRead: true})
		assert.Equal(t, unread[:1], listIDs(true))
		assert.ElementsMatch(t, unread[1:], listIDs(false))

		markRead(advisor.MarkInsightsReadBody{Filters: &advisor.MarkInsightsReadParamsBodyFilters{RunID: runID}, IsRead: true})
		assert.ElementsMatch(t, unread, listIDs(true))
		assert.Empty(t, listIDs(false))

		markRead(advisor.MarkInsightsReadBody{Ids: unread, IsRead: false})
		assert.ElementsMatch(t, unread, listIDs(false))

		resp, err := advisorClient.Default.AdvisorService.MarkInsightsRead(&advisor.MarkInsightsReadParams{
			Body:    advisor.MarkInsightsReadBody{IsRead: true},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Either ids or filters must be provided.")
		assert.Nil(t, resp)
	})

	t.Run("list runs", func(t *testing.T) {
		startedAt := func(id string) time.Time {
			resp, err := advisorClient.Default.AdvisorService.ListRuns(&advisor.ListRunsParams{
				RunID:   &id,
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)
			require.Len(t, resp.Payload.Results, 1)
			return time.Time(resp.Payload.Results[0].StartedAt)
		}
		listRunIDs := func(params *advisor.ListRunsParams) []string {
			params.Context = pmmapitests.Context
			resp, err := advisorClient.Default.AdvisorService.ListRuns(params)
			require.NoError(t, err)
			ids := make([]string, 0, len(resp.Payload.Results))
			for _, r := range resp.Payload.Results {
				ids = append(ids, r.ID)
			}
			return ids
		}

		// query parameters truncate the microseconds of started_at to milliseconds
		from := strfmt.DateTime(startedAt(firstRunID))
		to := strfmt.DateTime(startedAt(runID).Add(time.Millisecond))
		user := advisor.ListRunsOKBodyResultsItems0TriggeredByADVISORCHECKTRIGGEREDBYUSER
		scheduler := advisor.ListRunsOKBodyResultsItems0TriggeredByADVISORCHECKTRIGGEREDBYSCHEDULER

		t.Run("pagination", func(t *testing.T) {
			// newest first
			for page, want := range []string{runID, firstRunID} {
				resp, err := advisorClient.Default.AdvisorService.ListRuns(&advisor.ListRunsParams{
					TriggeredBy: &user,
					From:        &from,
					To:          &to,
					PageSize:    new(int32(1)),
					PageIndex:   new(int32(page)),
					Context:     pmmapitests.Context,
				})
				require.NoError(t, err)
				assert.Equal(t, int32(2), resp.Payload.TotalItems)
				assert.Equal(t, int32(2), resp.Payload.TotalPages)
				require.Len(t, resp.Payload.Results, 1)
				assert.Equal(t, want, resp.Payload.Results[0].ID)
			}
		})

		t.Run("triggered by", func(t *testing.T) {
			assert.Equal(t, []string{runID, firstRunID}, listRunIDs(&advisor.ListRunsParams{TriggeredBy: &user, From: &from, To: &to}))

			ids := listRunIDs(&advisor.ListRunsParams{TriggeredBy: &scheduler, From: &from, To: &to})
			assert.NotContains(t, ids, runID)
			assert.NotContains(t, ids, firstRunID)
		})

		t.Run("time range", func(t *testing.T) {
			between := strfmt.DateTime(startedAt(runID).Add(-time.Millisecond))
			assert.Equal(t, []string{runID}, listRunIDs(&advisor.ListRunsParams{TriggeredBy: &user, From: &between, To: &to}))
			assert.Equal(t, []string{firstRunID}, listRunIDs(&advisor.ListRunsParams{TriggeredBy: &user, From: &from, To: &between}))
		})
	})
}

func TestSendTestAdvisorNotification(t *testing.T) {
	// api-tests run against a PMM Server without SMTP, so no email is sent
	resp, err := advisorClient.Default.AdvisorService.SendTestAdvisorNotification(&advisor.SendTestAdvisorNotificationParams{
		Body: advisor.SendTestAdvisorNotificationBody{
			EmailAddresses: []string{"dba@example.com"},
			Severities:     severities("SEVERITY_CRITICAL"),
		},
		Context: pmmapitests.Context,
	})
	pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, "SMTP is not configured in PMM Server: GF_SMTP_ENABLED is not set.")
	assert.Nil(t, resp)
}

// addAdvisorTargetService adds a MySQL service run by PMM Server's own pmm-agent,
// which makes it a target of Advisor checks; nothing listens on its address.
func addAdvisorTargetService(t *testing.T) (string, string) {
	t.Helper()

	serviceName := pmmapitests.TestString(t, "advisor-target")
	resp, err := managementClient.Default.ManagementService.AddService(&mservice.AddServiceParams{
		Body: mservice.AddServiceBody{
			Mysql: &mservice.AddServiceParamsBodyMysql{
				NodeID:              "pmm-server",
				PMMAgentID:          "pmm-server",
				ServiceName:         serviceName,
				Address:             "127.0.0.1",
				Port:                3306,
				Username:            "username",
				SkipConnectionCheck: true,
			},
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	serviceID := resp.Payload.Mysql.Service.ServiceID
	t.Cleanup(func() { pmmapitests.RemoveServices(t, serviceID) })

	return serviceID, serviceName
}

// customCheckName returns a unique name in the namespace reserved for user checks.
func customCheckName(t *testing.T) string {
	t.Helper()

	return "custom_" + strings.ReplaceAll(pmmapitests.TestString(t, "check"), "-", "_")
}

// customCheckBody defines a check that queries only PMM's own metrics, so it
// yields the same results whether or not the service's database is reachable.
func customCheckBody(name string) *advisor.CreateAdvisorCheckParamsBodyCheck {
	return &advisor.CreateAdvisorCheckParamsBodyCheck{
		Name:        name,
		Summary:     "API test check",
		Description: "Created by an API test.",
		Category:    "API tests",
		Technology:  new(advisor.CreateAdvisorCheckParamsBodyCheckTechnologyADVISORCHECKTECHNOLOGYMYSQL),
		Queries:     []*advisor.CreateAdvisorCheckParamsBodyCheckQueriesItems0{{Type: "METRICS_INSTANT", Query: "vector(1)"}},
		Script:      customCheckScript,
	}
}

// updateCheckBody is the definition of customCheckBody as UpdateAdvisorCheck takes it, without a name.
func updateCheckBody() *advisor.UpdateAdvisorCheckParamsBodyCheck {
	return &advisor.UpdateAdvisorCheckParamsBodyCheck{
		Summary:     "API test check",
		Description: "Created by an API test.",
		Category:    "API tests",
		Technology:  new(advisor.UpdateAdvisorCheckParamsBodyCheckTechnologyADVISORCHECKTECHNOLOGYMYSQL),
		Queries:     []*advisor.UpdateAdvisorCheckParamsBodyCheckQueriesItems0{{Type: "METRICS_INSTANT", Query: "vector(1)"}},
		Script:      customCheckScript,
	}
}

// testCheckBody is the definition of customCheckBody with the given script, as TestAdvisorCheck takes it.
func testCheckBody(name, script string) *advisor.TestAdvisorCheckParamsBodyCheck {
	return &advisor.TestAdvisorCheckParamsBodyCheck{
		Name:        name,
		Summary:     "API test check",
		Description: "Created by an API test.",
		Category:    "API tests",
		Technology:  new(advisor.TestAdvisorCheckParamsBodyCheckTechnologyADVISORCHECKTECHNOLOGYMYSQL),
		Queries:     []*advisor.TestAdvisorCheckParamsBodyCheckQueriesItems0{{Type: "METRICS_INSTANT", Query: "vector(1)"}},
		Script:      script,
	}
}

// createCustomCheck creates the check of customCheckBody and deletes it when the test ends.
func createCustomCheck(t *testing.T, name string) *advisor.CreateAdvisorCheckOKBodyCheck {
	t.Helper()

	resp, err := advisorClient.Default.AdvisorService.CreateAdvisorCheck(&advisor.CreateAdvisorCheckParams{
		Body:    advisor.CreateAdvisorCheckBody{Check: customCheckBody(name)},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		// the test may have deleted it already
		_, _ = advisorClient.Default.AdvisorService.DeleteAdvisorCheck(&advisor.DeleteAdvisorCheckParams{
			Name:    name,
			Context: pmmapitests.Context,
		})
	})

	return resp.Payload.Check
}

// startAdvisorRun runs a single check against a single service and waits for the run to end.
func startAdvisorRun(t *testing.T, checkName, serviceID string) string {
	t.Helper()

	waitForNoActiveRun(t)
	resp, err := advisorClient.Default.AdvisorService.StartAdvisorChecks(&advisor.StartAdvisorChecksParams{
		Body: advisor.StartAdvisorChecksBody{
			Names:      []string{checkName},
			ServiceIds: []string{serviceID},
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	waitForNoActiveRun(t)

	return resp.Payload.RunID
}

// waitForNoActiveRun waits until no Advisor run is queued or running, so a new one can start.
func waitForNoActiveRun(t *testing.T) {
	t.Helper()

	require.Eventually(t, func() bool {
		resp, err := advisorClient.Default.AdvisorService.ListRuns(&advisor.ListRunsParams{
			PageSize: new(int32(10)),
			Context:  pmmapitests.Context,
		})
		if err != nil {
			return false
		}
		for _, run := range resp.Payload.Results {
			if run.Status == nil {
				continue
			}
			switch *run.Status {
			case advisor.ListRunsOKBodyResultsItems0StatusADVISORRUNSTATUSQUEUED,
				advisor.ListRunsOKBodyResultsItems0StatusADVISORRUNSTATUSRUNNING:
				return false
			}
		}
		return true
	}, 5*time.Minute, time.Second)
}

func toggleAdvisorChecks(t *testing.T, enable bool) {
	t.Helper()

	res, err := serverClient.Default.ServerService.ChangeSettings(&server_service.ChangeSettingsParams{
		Body: server_service.ChangeSettingsBody{
			EnableAdvisor: new(enable),
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	require.Equal(t, enable, res.Payload.Settings.AdvisorEnabled)

	if enable {
		// It takes some time to load check files
		time.Sleep(time.Second)
	}
}

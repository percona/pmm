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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	advisorClient "github.com/percona/pmm/api/advisors/v1/json/client"
	advisor "github.com/percona/pmm/api/advisors/v1/json/client/advisor_service"
	inventoryClient "github.com/percona/pmm/api/inventory/v1/json/client"
	inventory "github.com/percona/pmm/api/inventory/v1/json/client/services_service"
	managementClient "github.com/percona/pmm/api/management/v1/json/client"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
	serverClient "github.com/percona/pmm/api/server/v1/json/client"
	"github.com/percona/pmm/api/server/v1/json/client/server_service"
)

const (
	scheduleEmailSettings = advisor.CreateAdvisorScheduleBodyEmailModeADVISORSCHEDULEEMAILMODESETTINGS
	scheduleEmailCustom   = advisor.CreateAdvisorScheduleBodyEmailModeADVISORSCHEDULEEMAILMODECUSTOM
	scheduleEmailOff      = advisor.CreateAdvisorScheduleBodyEmailModeADVISORSCHEDULEEMAILMODEOFF
)

func TestAdvisorSchedules(t *testing.T) {
	hour := quietHour()
	at := func(minute int) string { return fmt.Sprintf("%02d:%02d", hour, minute) }

	name := scheduleName("crud")
	created := createSchedule(t, advisor.CreateAdvisorScheduleBody{
		Name:            "  " + name + "  ",
		DaysOfWeek:      []int32{5, 1, 3, 1},
		TimeOfDay:       at(40),
		CheckNames:      []string{"custom_b", "", "custom_a", "custom_b"},
		ServiceIds:      []string{"service-2", "service-1", "service-2"},
		EmailMode:       new(scheduleEmailCustom),
		EmailAddresses:  []string{"b@example.com", "a@example.com", "b@example.com"},
		EmailSeverities: severities("SEVERITY_WARNING", "SEVERITY_CRITICAL", "SEVERITY_WARNING"),
	})

	t.Run("create normalizes the schedule", func(t *testing.T) {
		assert.NotEmpty(t, created.ScheduleID)
		assert.Equal(t, name, created.Name)
		assert.True(t, created.Enabled, "absent enabled means enabled")
		assert.Equal(t, []int32{1, 3, 5}, created.DaysOfWeek)
		assert.Equal(t, at(40), created.TimeOfDay)
		assert.Equal(t, []string{"custom_b", "custom_a"}, created.CheckNames)
		assert.Equal(t, []string{"service-2", "service-1"}, created.ServiceIds)
		assert.Equal(t, scheduleEmailCustom, *created.EmailMode)
		assert.Equal(t, []string{"b@example.com", "a@example.com"}, created.EmailAddresses)
		assert.Equal(t, severities("SEVERITY_CRITICAL", "SEVERITY_WARNING"), created.EmailSeverities)
		assert.Nil(t, created.LastRun)
		assert.Equal(t, time.Time(created.CreatedAt), time.Time(created.UpdatedAt))
		assert.Equal(t, nextOccurrence(time.Time(created.UpdatedAt), created.DaysOfWeek, created.TimeOfDay), time.Time(created.NextRunAt))
	})

	t.Run("list", func(t *testing.T) {
		prefix := scheduleName("list")
		b := createSchedule(t, advisor.CreateAdvisorScheduleBody{
			Name:       prefix + "-b",
			DaysOfWeek: []int32{0, 1, 2, 3, 4, 5, 6},
			TimeOfDay:  at(20),
			EmailMode:  new(scheduleEmailSettings),
		})
		a := createSchedule(t, advisor.CreateAdvisorScheduleBody{
			Name:       prefix + "-a",
			Enabled:    new(false),
			DaysOfWeek: []int32{2},
			TimeOfDay:  at(20),
			EmailMode:  new(scheduleEmailOff),
		})

		schedules := listSchedules(t)
		assert.True(t, slices.IsSortedFunc(schedules, func(x, y *advisor.ListAdvisorSchedulesOKBodySchedulesItems0) int {
			return strings.Compare(x.TimeOfDay, y.TimeOfDay)
		}), "schedules must be ordered by time of day")
		var ids []string
		for _, s := range schedules {
			if slices.Contains([]string{a.ScheduleID, b.ScheduleID, created.ScheduleID}, s.ScheduleID) {
				ids = append(ids, s.ScheduleID)
			}
		}
		assert.Equal(t, []string{a.ScheduleID, b.ScheduleID, created.ScheduleID}, ids, "same time of day orders by name")

		listedA := findSchedule(t, schedules, a.ScheduleID)
		assert.False(t, listedA.Enabled)
		assert.True(t, time.Time(listedA.NextRunAt).IsZero(), "a disabled schedule has no next run")

		listedB := findSchedule(t, schedules, b.ScheduleID)
		assert.True(t, listedB.Enabled)
		assert.Empty(t, listedB.CheckNames)
		assert.Empty(t, listedB.ServiceIds)
		assert.Equal(t, nextOccurrence(time.Time(listedB.UpdatedAt), listedB.DaysOfWeek, listedB.TimeOfDay), time.Time(listedB.NextRunAt))

		listed := findSchedule(t, schedules, created.ScheduleID)
		assert.Equal(t, created.Name, listed.Name)
		assert.Equal(t, created.DaysOfWeek, listed.DaysOfWeek)
		assert.Equal(t, created.CheckNames, listed.CheckNames)
		assert.Equal(t, created.ServiceIds, listed.ServiceIds)
		assert.Equal(t, created.EmailAddresses, listed.EmailAddresses)
		assert.Equal(t, created.EmailSeverities, listed.EmailSeverities)
		assert.Equal(t, created.NextRunAt, listed.NextRunAt)
	})

	t.Run("update replaces the whole schedule", func(t *testing.T) {
		renamed := scheduleName("renamed")
		updated := updateSchedule(t, created.ScheduleID, advisor.UpdateAdvisorScheduleBody{
			Name:            renamed,
			Enabled:         new(false),
			DaysOfWeek:      []int32{6, 0},
			TimeOfDay:       at(50),
			EmailMode:       new(scheduleEmailOff),
			EmailAddresses:  []string{"a@example.com"},
			EmailSeverities: severities("SEVERITY_CRITICAL"),
		})
		assert.Equal(t, created.ScheduleID, updated.ScheduleID)
		assert.Equal(t, renamed, updated.Name)
		assert.False(t, updated.Enabled)
		assert.Equal(t, []int32{0, 6}, updated.DaysOfWeek)
		assert.Equal(t, at(50), updated.TimeOfDay)
		assert.Empty(t, updated.CheckNames)
		assert.Empty(t, updated.ServiceIds)
		assert.Equal(t, scheduleEmailOff, *updated.EmailMode)
		assert.Empty(t, updated.EmailAddresses)
		assert.Empty(t, updated.EmailSeverities)
		assert.True(t, time.Time(updated.NextRunAt).IsZero(), "a disabled schedule has no next run")
		assert.Equal(t, time.Time(created.CreatedAt), time.Time(updated.CreatedAt))
		assert.True(t, time.Time(updated.UpdatedAt).After(time.Time(created.UpdatedAt)))

		listed := findSchedule(t, listSchedules(t), created.ScheduleID)
		assert.Equal(t, renamed, listed.Name)
		assert.False(t, listed.Enabled)
		assert.Equal(t, []int32{0, 6}, listed.DaysOfWeek)

		reenabled := updateSchedule(t, created.ScheduleID, advisor.UpdateAdvisorScheduleBody{
			Name:       renamed,
			DaysOfWeek: []int32{0, 6},
			TimeOfDay:  at(50),
			EmailMode:  new(scheduleEmailOff),
		})
		assert.True(t, reenabled.Enabled, "absent enabled means enabled")
		assert.Equal(t, nextOccurrence(time.Time(reenabled.UpdatedAt), reenabled.DaysOfWeek, reenabled.TimeOfDay), time.Time(reenabled.NextRunAt))
	})

	t.Run("delete", func(t *testing.T) {
		_, err := advisorClient.Default.AdvisorService.DeleteAdvisorSchedule(&advisor.DeleteAdvisorScheduleParams{
			ScheduleID: created.ScheduleID,
			Context:    pmmapitests.Context,
		})
		require.NoError(t, err)

		for _, s := range listSchedules(t) {
			assert.NotEqual(t, created.ScheduleID, s.ScheduleID, "the schedule is still listed")
		}
	})

	t.Run("unknown schedule", func(t *testing.T) {
		id := "unknown-schedule-id"

		updated, err := advisorClient.Default.AdvisorService.UpdateAdvisorSchedule(&advisor.UpdateAdvisorScheduleParams{
			ScheduleID: id,
			Body: advisor.UpdateAdvisorScheduleBody{
				Name:       "unknown",
				DaysOfWeek: []int32{1},
				TimeOfDay:  at(0),
				EmailMode:  new(scheduleEmailSettings),
			},
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Advisor schedule with ID %s not found.", id)
		assert.Nil(t, updated)

		deleted, err := advisorClient.Default.AdvisorService.DeleteAdvisorSchedule(&advisor.DeleteAdvisorScheduleParams{
			ScheduleID: id,
			Context:    pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Advisor schedule with ID %s not found.", id)
		assert.Nil(t, deleted)

		run, err := advisorClient.Default.AdvisorService.RunAdvisorSchedule(&advisor.RunAdvisorScheduleParams{
			ScheduleID: id,
			Context:    pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Advisor schedule with ID %s not found.", id)
		assert.Nil(t, run)
	})
}

func TestAdvisorScheduleValidation(t *testing.T) {
	hour := quietHour()
	valid := func() advisor.CreateAdvisorScheduleBody {
		return advisor.CreateAdvisorScheduleBody{
			Name:       scheduleName("valid"),
			DaysOfWeek: []int32{1},
			TimeOfDay:  fmt.Sprintf("%02d:10", hour),
			EmailMode:  new(scheduleEmailSettings),
		}
	}
	addresses := func(n int) []string {
		res := make([]string, 0, n)
		for i := range n {
			res = append(res, fmt.Sprintf("dba%d@example.com", i))
		}
		return res
	}

	t.Run("invalid creates are rejected", func(t *testing.T) {
		before := len(listSchedules(t))
		const badTime = "invalid CreateAdvisorScheduleRequest.TimeOfDay: value does not match regex pattern"

		for _, tc := range []struct {
			name    string
			change  func(b *advisor.CreateAdvisorScheduleBody)
			message string
		}{
			{
				"empty name", func(b *advisor.CreateAdvisorScheduleBody) { b.Name = "" },
				"invalid CreateAdvisorScheduleRequest.Name: value length must be between 1 and 100 runes, inclusive",
			},
			{
				"blank name", func(b *advisor.CreateAdvisorScheduleBody) { b.Name = "   " },
				"Advisor schedule name is required.",
			},
			{
				"name over 100 characters", func(b *advisor.CreateAdvisorScheduleBody) { b.Name = strings.Repeat("n", 101) },
				"invalid CreateAdvisorScheduleRequest.Name: value length must be between 1 and 100 runes, inclusive",
			},
			{
				"no days", func(b *advisor.CreateAdvisorScheduleBody) { b.DaysOfWeek = nil },
				"invalid CreateAdvisorScheduleRequest.DaysOfWeek: value must contain at least 1 item(s)",
			},
			{
				"day after Saturday", func(b *advisor.CreateAdvisorScheduleBody) { b.DaysOfWeek = []int32{1, 7} },
				"invalid CreateAdvisorScheduleRequest.DaysOfWeek[1]: value must be inside range [0, 6]",
			},
			{
				"day before Sunday", func(b *advisor.CreateAdvisorScheduleBody) { b.DaysOfWeek = []int32{-1} },
				"invalid CreateAdvisorScheduleRequest.DaysOfWeek[0]: value must be inside range [0, 6]",
			},
			{"hour 24", func(b *advisor.CreateAdvisorScheduleBody) { b.TimeOfDay = "24:00" }, badTime},
			{"single-digit hour", func(b *advisor.CreateAdvisorScheduleBody) { b.TimeOfDay = "6:00" }, badTime},
			{"minute 60", func(b *advisor.CreateAdvisorScheduleBody) { b.TimeOfDay = "06:60" }, badTime},
			{"no time", func(b *advisor.CreateAdvisorScheduleBody) { b.TimeOfDay = "" }, badTime},
			{
				"no email mode", func(b *advisor.CreateAdvisorScheduleBody) { b.EmailMode = nil },
				"invalid CreateAdvisorScheduleRequest.EmailMode: value must not be in list [ADVISOR_SCHEDULE_EMAIL_MODE_UNSPECIFIED]",
			},
			{
				"custom email without recipients", func(b *advisor.CreateAdvisorScheduleBody) { b.EmailMode = new(scheduleEmailCustom) },
				"At least one email address is required for custom recipients.",
			},
			{"more than 20 recipients", func(b *advisor.CreateAdvisorScheduleBody) {
				b.EmailMode = new(scheduleEmailCustom)
				b.EmailAddresses = addresses(21)
			}, "Invalid email addresses: at most 20 addresses are allowed, got 21."},
			{"invalid recipient", func(b *advisor.CreateAdvisorScheduleBody) {
				b.EmailMode = new(scheduleEmailCustom)
				b.EmailAddresses = []string{"dba@example.com", "not-an-email"}
			}, "Invalid email addresses: invalid address 'not-an-email'."},
			{
				"unsupported severity", func(b *advisor.CreateAdvisorScheduleBody) { b.EmailSeverities = severities("SEVERITY_NOTICE") },
				"Unsupported email severity notice: use critical, error, warning or info.",
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				body := valid()
				tc.change(&body)
				resp, err := advisorClient.Default.AdvisorService.CreateAdvisorSchedule(&advisor.CreateAdvisorScheduleParams{
					Body:    body,
					Context: pmmapitests.Context,
				})
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, tc.message)
				assert.Nil(t, resp)
			})
		}

		assert.Len(t, listSchedules(t), before, "a rejected schedule was stored")
	})

	t.Run("limits are inclusive", func(t *testing.T) {
		body := valid()
		body.Name = (body.Name + strings.Repeat("n", 100))[:100]
		body.EmailMode = new(scheduleEmailCustom)
		body.EmailAddresses = addresses(20)
		s := createSchedule(t, body)
		assert.Len(t, []rune(s.Name), 100)
		assert.Len(t, s.EmailAddresses, 20)
	})

	t.Run("invalid updates are rejected", func(t *testing.T) {
		s := createSchedule(t, valid())

		for _, tc := range []struct {
			name    string
			change  func(b *advisor.UpdateAdvisorScheduleBody)
			message string
		}{
			{
				"bad time", func(b *advisor.UpdateAdvisorScheduleBody) { b.TimeOfDay = "24:00" },
				"invalid UpdateAdvisorScheduleRequest.TimeOfDay: value does not match regex pattern",
			},
			{
				"custom email without recipients", func(b *advisor.UpdateAdvisorScheduleBody) { b.EmailMode = new(scheduleEmailCustom) },
				"At least one email address is required for custom recipients.",
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				body := advisor.UpdateAdvisorScheduleBody{
					Name:       s.Name,
					DaysOfWeek: s.DaysOfWeek,
					TimeOfDay:  s.TimeOfDay,
					EmailMode:  s.EmailMode,
				}
				tc.change(&body)
				resp, err := advisorClient.Default.AdvisorService.UpdateAdvisorSchedule(&advisor.UpdateAdvisorScheduleParams{
					ScheduleID: s.ScheduleID,
					Body:       body,
					Context:    pmmapitests.Context,
				})
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, tc.message)
				assert.Nil(t, resp)
			})
		}

		listed := findSchedule(t, listSchedules(t), s.ScheduleID)
		assert.Equal(t, s.TimeOfDay, listed.TimeOfDay)
		assert.Equal(t, s.EmailMode, listed.EmailMode)
	})

	t.Run("names are unique ignoring case", func(t *testing.T) {
		first := createSchedule(t, valid())
		second := createSchedule(t, valid())

		upper := valid()
		upper.Name = strings.ToUpper(first.Name)
		resp, err := advisorClient.Default.AdvisorService.CreateAdvisorSchedule(&advisor.CreateAdvisorScheduleParams{
			Body:    upper,
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 409, codes.AlreadyExists, "Advisor schedule '%s' already exists.", upper.Name)
		assert.Nil(t, resp)

		rename := func(id, name string) (*advisor.UpdateAdvisorScheduleOK, error) {
			return advisorClient.Default.AdvisorService.UpdateAdvisorSchedule(&advisor.UpdateAdvisorScheduleParams{
				ScheduleID: id,
				Body: advisor.UpdateAdvisorScheduleBody{
					Name:       name,
					DaysOfWeek: second.DaysOfWeek,
					TimeOfDay:  second.TimeOfDay,
					EmailMode:  second.EmailMode,
				},
				Context: pmmapitests.Context,
			})
		}

		updated, err := rename(second.ScheduleID, strings.ToUpper(first.Name))
		pmmapitests.AssertAPIErrorf(t, err, 409, codes.AlreadyExists, "Advisor schedule '%s' already exists.", strings.ToUpper(first.Name))
		assert.Nil(t, updated)
		assert.Equal(t, second.Name, findSchedule(t, listSchedules(t), second.ScheduleID).Name)

		updated, err = rename(first.ScheduleID, strings.ToUpper(first.Name))
		require.NoError(t, err, "a schedule can change the case of its own name")
		assert.Equal(t, strings.ToUpper(first.Name), updated.Payload.Schedule.Name)
	})

	t.Run("email modes", func(t *testing.T) {
		t.Run("off drops recipients and severities", func(t *testing.T) {
			body := valid()
			body.EmailMode = new(scheduleEmailOff)
			body.EmailAddresses = []string{"dba@example.com"}
			body.EmailSeverities = severities("SEVERITY_CRITICAL")
			s := createSchedule(t, body)
			assert.Equal(t, scheduleEmailOff, *s.EmailMode)
			assert.Empty(t, s.EmailAddresses)
			assert.Empty(t, s.EmailSeverities)
		})

		t.Run("settings drops recipients", func(t *testing.T) {
			body := valid()
			body.EmailAddresses = []string{"dba@example.com"}
			body.EmailSeverities = severities("SEVERITY_ERROR", "SEVERITY_CRITICAL")
			s := createSchedule(t, body)
			assert.Equal(t, scheduleEmailSettings, *s.EmailMode)
			assert.Empty(t, s.EmailAddresses)
			assert.Equal(t, severities("SEVERITY_CRITICAL", "SEVERITY_ERROR"), s.EmailSeverities)
		})

		t.Run("no severities means the settings severities", func(t *testing.T) {
			for _, mode := range []string{scheduleEmailSettings, scheduleEmailCustom} {
				body := valid()
				body.EmailMode = new(mode)
				body.EmailAddresses = []string{"dba@example.com"}
				s := createSchedule(t, body)
				assert.Empty(t, s.EmailSeverities, mode)
				assert.Empty(t, findSchedule(t, listSchedules(t), s.ScheduleID).EmailSeverities, mode)
			}
		})

		t.Run("a test email needs a severity", func(t *testing.T) {
			resp, err := advisorClient.Default.AdvisorService.SendTestAdvisorNotification(&advisor.SendTestAdvisorNotificationParams{
				Body:    advisor.SendTestAdvisorNotificationBody{EmailAddresses: []string{"dba@example.com"}},
				Context: pmmapitests.Context,
			})
			pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "At least one severity is required.")
			assert.Nil(t, resp)
		})
	})
}

func TestAdvisorSchedulePermissions(t *testing.T) {
	toggleAdvisorChecks(t, true)
	t.Cleanup(func() { RestoreSettingsDefaults(t) })

	serviceID, _ := addAdvisorTargetService(t)
	checkName := customCheckName(t)
	createCustomCheck(t, checkName)

	ts := strconv.FormatInt(time.Now().UnixNano(), 10)
	editor := "advisor-editor-" + ts
	editorID := createUserWithRole(t, editor, "Editor")
	t.Cleanup(func() { deleteUser(t, editorID) })
	viewer := "advisor-viewer-" + ts
	viewerID := createUserWithRole(t, viewer, "Viewer")
	t.Cleanup(func() { deleteUser(t, viewerID) })

	body := advisor.CreateAdvisorScheduleBody{
		Name:       scheduleName("rbac"),
		Enabled:    new(false),
		DaysOfWeek: []int32{1},
		TimeOfDay:  fmt.Sprintf("%02d:00", quietHour()),
		CheckNames: []string{checkName},
		ServiceIds: []string{serviceID},
		EmailMode:  new(scheduleEmailOff),
	}
	testEmail := advisor.SendTestAdvisorNotificationBody{
		EmailAddresses: []string{"dba@example.com"},
		Severities:     severities("SEVERITY_CRITICAL"),
	}

	t.Run("editor", func(t *testing.T) {
		code, b := requestAs(t, editor, http.MethodGet, "/v1/advisors/schedules", nil)
		require.Equal(t, http.StatusOK, code, "%s", b)

		code, b = requestAs(t, editor, http.MethodPost, "/v1/advisors/schedules", body)
		require.Equal(t, http.StatusOK, code, "%s", b)
		var created advisor.CreateAdvisorScheduleOKBody
		require.NoError(t, json.Unmarshal(b, &created))
		id := created.Schedule.ScheduleID
		t.Cleanup(func() { deleteScheduleIfExists(id) })

		update := advisor.UpdateAdvisorScheduleBody{
			Name:       body.Name + "-renamed",
			Enabled:    new(false),
			DaysOfWeek: body.DaysOfWeek,
			TimeOfDay:  body.TimeOfDay,
			CheckNames: body.CheckNames,
			ServiceIds: body.ServiceIds,
			EmailMode:  body.EmailMode,
		}
		code, b = requestAs(t, editor, http.MethodPut, "/v1/advisors/schedules/"+id, update)
		require.Equal(t, http.StatusOK, code, "%s", b)
		assert.Equal(t, update.Name, findSchedule(t, listSchedules(t), id).Name)

		waitForNoActiveRun(t)
		code, b = requestAs(t, editor, http.MethodPost, "/v1/advisors/schedules/"+id+":run", nil)
		require.Equal(t, http.StatusOK, code, "%s", b)
		var run advisor.RunAdvisorScheduleOKBody
		require.NoError(t, json.Unmarshal(b, &run))
		assert.NotEmpty(t, run.RunID)
		waitForNoActiveRun(t)

		code, b = requestAs(t, editor, http.MethodDelete, "/v1/advisors/schedules/"+id, nil)
		require.Equal(t, http.StatusOK, code, "%s", b)
		for _, s := range listSchedules(t) {
			assert.NotEqual(t, id, s.ScheduleID, "the schedule is still listed")
		}

		code, b = requestAs(t, editor, http.MethodGet, "/v1/advisors/services", nil)
		assert.Equal(t, http.StatusOK, code, "%s", b)

		code, b = requestAs(t, editor, http.MethodGet, "/v1/advisors/notificationSettings", nil)
		assert.Equal(t, http.StatusOK, code, "%s", b)

		// the server has no SMTP, so the request fails only after it passes authorization
		code, b = requestAs(t, editor, http.MethodPost, "/v1/advisors/notifications:test", testEmail)
		assert.Equal(t, http.StatusBadRequest, code, "%s", b)
		var apiErr struct {
			Code    codes.Code `json:"code"`
			Message string     `json:"message"`
		}
		require.NoError(t, json.Unmarshal(b, &apiErr))
		assert.Equal(t, codes.FailedPrecondition, apiErr.Code)
		assert.Equal(t, "SMTP is not configured in PMM Server: GF_SMTP_ENABLED is not set.", apiErr.Message)
	})

	t.Run("viewer", func(t *testing.T) {
		own := body
		own.Name = scheduleName("rbac")
		s := createSchedule(t, own)
		path := "/v1/advisors/schedules/" + s.ScheduleID

		for _, r := range []struct {
			method string
			path   string
			body   any
		}{
			{http.MethodGet, "/v1/advisors/schedules", nil},
			{http.MethodPost, "/v1/advisors/schedules", body},
			{http.MethodPut, path, body},
			{http.MethodPost, path + ":run", nil},
			{http.MethodDelete, path, nil},
			{http.MethodGet, "/v1/advisors/services", nil},
			{http.MethodGet, "/v1/advisors/notificationSettings", nil},
			{http.MethodPost, "/v1/advisors/notifications:test", testEmail},
		} {
			t.Run(r.method+" "+strings.Replace(r.path, s.ScheduleID, "{schedule_id}", 1), func(t *testing.T) {
				code, b := requestAs(t, viewer, r.method, r.path, r.body)
				assert.Equal(t, http.StatusForbidden, code, "%s", b)
			})
		}

		listed := findSchedule(t, listSchedules(t), s.ScheduleID)
		assert.Equal(t, s.Name, listed.Name)
		assert.Nil(t, listed.LastRun)
	})
}

func TestRunAdvisorSchedule(t *testing.T) {
	toggleAdvisorChecks(t, true)
	t.Cleanup(func() { RestoreSettingsDefaults(t) })

	serviceID, _ := addAdvisorTargetService(t)
	checkName := customCheckName(t)
	createCustomCheck(t, checkName)
	hour := quietHour()

	s := createSchedule(t, advisor.CreateAdvisorScheduleBody{
		Name:       scheduleName("run-now"),
		DaysOfWeek: []int32{1},
		TimeOfDay:  fmt.Sprintf("%02d:00", hour),
		CheckNames: []string{checkName},
		ServiceIds: []string{serviceID},
		EmailMode:  new(scheduleEmailOff),
	})
	stalledCheck, stalledServiceID := addStallingCheck(t)
	blocker := createSchedule(t, advisor.CreateAdvisorScheduleBody{
		Name:       scheduleName("blocker"),
		DaysOfWeek: []int32{1},
		TimeOfDay:  fmt.Sprintf("%02d:30", hour),
		CheckNames: []string{stalledCheck},
		ServiceIds: []string{stalledServiceID},
		EmailMode:  new(scheduleEmailOff),
	})

	var runID string
	t.Run("run now", func(t *testing.T) {
		waitForNoActiveRun(t)
		runID = runSchedule(t, s.ScheduleID)
		waitForNoActiveRun(t)

		run := getRun(t, runID)
		assert.Equal(t, advisor.ListRunsOKBodyResultsItems0StatusADVISORRUNSTATUSCOMPLETED, *run.Status)
		assert.Equal(t, advisor.ListRunsOKBodyResultsItems0TriggeredByADVISORCHECKTRIGGEREDBYUSER, *run.TriggeredBy)
		assert.Equal(t, s.ScheduleID, run.ScheduleID)
		assert.Equal(t, s.Name, run.ScheduleName)
		assert.True(t, time.Time(run.ScheduledFor).IsZero(), "a run started by a user has no scheduled time")
		assert.Nil(t, run.WaitedForRun)
		assert.Equal(t, []string{checkName}, run.CheckNames)
		assert.Equal(t, []string{serviceID}, run.ServiceIds)
		assert.Equal(t, int32(3), run.FindingsCount)

		listed := findSchedule(t, listSchedules(t), s.ScheduleID)
		require.NotNil(t, listed.LastRun)
		assert.Equal(t, runID, listed.LastRun.ID)
	})

	var blockerRunID string
	t.Run("rejected while a run is active", func(t *testing.T) {
		waitForNoActiveRun(t)
		blockerRunID = runSchedule(t, blocker.ScheduleID)

		resp, err := advisorClient.Default.AdvisorService.RunAdvisorSchedule(&advisor.RunAdvisorScheduleParams{
			ScheduleID: s.ScheduleID,
			Context:    pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, "Advisor checks are already running (started ")
		assert.Nil(t, resp)

		started, err := advisorClient.Default.AdvisorService.StartAdvisorChecks(nil)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, "Advisor checks are already running (started ")
		assert.Nil(t, started)

		assert.Contains(t, []string{
			advisor.ListRunsOKBodyResultsItems0StatusADVISORRUNSTATUSQUEUED,
			advisor.ListRunsOKBodyResultsItems0StatusADVISORRUNSTATUSRUNNING,
		}, *getRun(t, blockerRunID).Status, "the blocking run ended before the requests were rejected")

		waitForNoActiveRun(t)
		run := getRun(t, blockerRunID)
		assert.Equal(t, advisor.ListRunsOKBodyResultsItems0StatusADVISORRUNSTATUSCOMPLETED, *run.Status)
		assert.Equal(t, int32(1), run.ErrorsCount)
	})

	t.Run("runs filtered by schedule", func(t *testing.T) {
		require.NotEmpty(t, runID)
		require.NotEmpty(t, blockerRunID)

		for id, want := range map[string]string{s.ScheduleID: runID, blocker.ScheduleID: blockerRunID} {
			resp, err := advisorClient.Default.AdvisorService.ListRuns(&advisor.ListRunsParams{
				ScheduleID: &id,
				Context:    pmmapitests.Context,
			})
			require.NoError(t, err)
			assert.Equal(t, int32(1), resp.Payload.TotalItems)
			require.Len(t, resp.Payload.Results, 1)
			assert.Equal(t, want, resp.Payload.Results[0].ID)
			assert.Equal(t, id, resp.Payload.Results[0].ScheduleID)
		}
	})

	t.Run("runs keep the name of a deleted schedule", func(t *testing.T) {
		require.NotEmpty(t, blockerRunID)

		_, err := advisorClient.Default.AdvisorService.DeleteAdvisorSchedule(&advisor.DeleteAdvisorScheduleParams{
			ScheduleID: blocker.ScheduleID,
			Context:    pmmapitests.Context,
		})
		require.NoError(t, err)

		run := getRun(t, blockerRunID)
		assert.Empty(t, run.ScheduleID)
		assert.Equal(t, blocker.Name, run.ScheduleName)

		resp, err := advisorClient.Default.AdvisorService.ListRuns(&advisor.ListRunsParams{
			ScheduleID: &blocker.ScheduleID,
			Context:    pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.Empty(t, resp.Payload.Results)
	})

	t.Run("with advisors disabled", func(t *testing.T) {
		toggleAdvisorChecks(t, false)

		resp, err := advisorClient.Default.AdvisorService.RunAdvisorSchedule(&advisor.RunAdvisorScheduleParams{
			ScheduleID: s.ScheduleID,
			Context:    pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, "advisor checks are disabled.")
		assert.Nil(t, resp)
	})
}

func TestDefaultAdvisorSchedule(t *testing.T) {
	toggleAdvisorChecks(t, true)
	waitForNoActiveRun(t)

	initial := listSchedules(t)
	t.Cleanup(func() {
		RestoreSettingsDefaults(t)
		deleteAllSchedules(t)
		for _, s := range initial {
			_, err := advisorClient.Default.AdvisorService.CreateAdvisorSchedule(&advisor.CreateAdvisorScheduleParams{
				Body: advisor.CreateAdvisorScheduleBody{
					Name:            s.Name,
					Enabled:         new(s.Enabled),
					DaysOfWeek:      s.DaysOfWeek,
					TimeOfDay:       s.TimeOfDay,
					CheckNames:      s.CheckNames,
					ServiceIds:      s.ServiceIds,
					EmailMode:       s.EmailMode,
					EmailAddresses:  s.EmailAddresses,
					EmailSeverities: s.EmailSeverities,
				},
				Context: pmmapitests.Context,
			})
			assert.NoError(t, err)
		}
	})

	t.Run("created when Advisors are turned on without schedules", func(t *testing.T) {
		deleteAllSchedules(t)
		toggleAdvisorChecks(t, false)
		total, latest := latestRun(t)

		toggleAdvisorChecks(t, true)
		schedules := listSchedules(t)
		require.Len(t, schedules, 1)
		s := schedules[0]
		assert.Equal(t, "All checks daily", s.Name)
		assert.True(t, s.Enabled)
		assert.Equal(t, []int32{0, 1, 2, 3, 4, 5, 6}, s.DaysOfWeek)
		assert.Equal(t, "06:00", s.TimeOfDay)
		assert.Empty(t, s.CheckNames)
		assert.Empty(t, s.ServiceIds)
		assert.Equal(t, scheduleEmailSettings, *s.EmailMode)
		assert.Empty(t, s.EmailAddresses)
		assert.Empty(t, s.EmailSeverities)
		assert.Equal(t, "06:00", time.Time(s.NextRunAt).UTC().Format("15:04"))

		totalAfter, latestAfter := latestRun(t)
		assert.Equal(t, total, totalAfter, "turning Advisors on must not start a run")
		assert.Equal(t, latest, latestAfter, "turning Advisors on must not start a run")
	})

	t.Run("not created when a schedule exists", func(t *testing.T) {
		deleteAllSchedules(t)
		own := createSchedule(t, advisor.CreateAdvisorScheduleBody{
			Name:       scheduleName("own"),
			Enabled:    new(false),
			DaysOfWeek: []int32{1},
			TimeOfDay:  fmt.Sprintf("%02d:00", quietHour()),
			EmailMode:  new(scheduleEmailSettings),
		})
		toggleAdvisorChecks(t, false)
		toggleAdvisorChecks(t, true)

		schedules := listSchedules(t)
		require.Len(t, schedules, 1)
		assert.Equal(t, own.ScheduleID, schedules[0].ScheduleID)
	})
}

func TestListAdvisorServices(t *testing.T) {
	mysqlID, mysqlName := addAdvisorTargetService(t)

	pgName := pmmapitests.TestString(t, "postgresql")
	pg, err := managementClient.Default.ManagementService.AddService(&mservice.AddServiceParams{
		Body: mservice.AddServiceBody{
			Postgresql: &mservice.AddServiceParamsBodyPostgresql{
				NodeID:              "pmm-server",
				PMMAgentID:          "pmm-server",
				ServiceName:         pgName,
				Address:             "127.0.0.1",
				Port:                15432,
				Username:            "username",
				Environment:         "api-test-env",
				Cluster:             "api-test-cluster",
				SkipConnectionCheck: true,
			},
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	pgID := pg.Payload.Postgresql.Service.ServiceID
	t.Cleanup(func() { pmmapitests.RemoveServices(t, pgID) })

	mongoName := pmmapitests.TestString(t, "mongodb")
	mongo, err := managementClient.Default.ManagementService.AddService(&mservice.AddServiceParams{
		Body: mservice.AddServiceBody{
			Mongodb: &mservice.AddServiceParamsBodyMongodb{
				NodeID:              "pmm-server",
				PMMAgentID:          "pmm-server",
				ServiceName:         mongoName,
				Address:             "127.0.0.1",
				Port:                27017,
				SkipConnectionCheck: true,
			},
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	mongoID := mongo.Payload.Mongodb.Service.ServiceID
	t.Cleanup(func() { pmmapitests.RemoveServices(t, mongoID) })

	resp, err := advisorClient.Default.AdvisorService.ListAdvisorServices(&advisor.ListAdvisorServicesParams{Context: pmmapitests.Context})
	require.NoError(t, err)
	listed := resp.Payload.Services
	assert.True(t, slices.IsSortedFunc(listed, func(a, b *advisor.ListAdvisorServicesOKBodyServicesItems0) int {
		return strings.Compare(a.ServiceName, b.ServiceName)
	}), "services must be ordered by name")

	for _, want := range []*advisor.ListAdvisorServicesOKBodyServicesItems0{
		{
			ServiceID:   mysqlID,
			ServiceName: mysqlName,
			Technology:  new(advisor.ListAdvisorServicesOKBodyServicesItems0TechnologyADVISORCHECKTECHNOLOGYMYSQL),
			NodeName:    "pmm-server",
		},
		{
			ServiceID:   pgID,
			ServiceName: pgName,
			Technology:  new(advisor.ListAdvisorServicesOKBodyServicesItems0TechnologyADVISORCHECKTECHNOLOGYPOSTGRESQL),
			NodeName:    "pmm-server",
			Environment: "api-test-env",
			Cluster:     "api-test-cluster",
		},
		{
			ServiceID:   mongoID,
			ServiceName: mongoName,
			Technology:  new(advisor.ListAdvisorServicesOKBodyServicesItems0TechnologyADVISORCHECKTECHNOLOGYMONGODB),
			NodeName:    "pmm-server",
		},
	} {
		assert.Contains(t, listed, want)
	}

	inv, err := inventoryClient.Default.ServicesService.ListServices(&inventory.ListServicesParams{Context: pmmapitests.Context})
	require.NoError(t, err)
	require.True(t, slices.ContainsFunc(inv.Payload.Postgresql, func(s *inventory.ListServicesOKBodyPostgresqlItems0) bool {
		return s.ServiceName == "pmm-server-postgresql"
	}), "PMM Server's own PostgreSQL service is not in the inventory")
	for _, s := range listed {
		assert.NotEqual(t, "pmm-server-postgresql", s.ServiceName, "PMM Server's own PostgreSQL must not be listed")
	}
}

func TestGetAdvisorNotificationSettings(t *testing.T) {
	get := func(t *testing.T) *advisor.GetAdvisorNotificationSettingsOKBody {
		t.Helper()
		resp, err := advisorClient.Default.AdvisorService.GetAdvisorNotificationSettings(&advisor.GetAdvisorNotificationSettingsParams{
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		return resp.Payload
	}
	change := func(t *testing.T, body server_service.ChangeSettingsBody) {
		t.Helper()
		_, err := serverClient.Default.ServerService.ChangeSettings(&server_service.ChangeSettingsParams{
			Body:    body,
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
	}

	settings, err := serverClient.Default.ServerService.GetSettings(&server_service.GetSettingsParams{Context: pmmapitests.Context})
	require.NoError(t, err)
	initial := settings.Payload.Settings
	t.Cleanup(func() {
		change(t, server_service.ChangeSettingsBody{
			EnableAdvisorNotifications:    new(initial.AdvisorNotificationsEnabled),
			AdvisorNotificationSeverities: initial.AdvisorNotificationSeverities,
			AdvisorNotificationEmailAddresses: &server_service.ChangeSettingsParamsBodyAdvisorNotificationEmailAddresses{
				Values: initial.AdvisorNotificationEmailAddresses,
			},
		})
	})

	t.Run("defaults", func(t *testing.T) {
		res := get(t)
		assert.Equal(t, initial.AdvisorNotificationsEnabled, res.Enabled)
		assert.Equal(t, initial.AdvisorNotificationEmailAddresses, res.EmailAddresses)
		assert.Equal(t, initial.AdvisorNotificationSeverities, res.Severities)
		// a fresh PMM Server has no severities set and emails the default ones
		assert.Equal(t, severities("SEVERITY_CRITICAL", "SEVERITY_ERROR"), res.Severities)
	})

	t.Run("reflects ChangeSettings", func(t *testing.T) {
		change(t, server_service.ChangeSettingsBody{
			EnableAdvisorNotifications:    new(true),
			AdvisorNotificationSeverities: severities("SEVERITY_WARNING", "SEVERITY_CRITICAL"),
			AdvisorNotificationEmailAddresses: &server_service.ChangeSettingsParamsBodyAdvisorNotificationEmailAddresses{
				Values: []string{"b@example.com", "a@example.com", "b@example.com"},
			},
		})
		res := get(t)
		assert.True(t, res.Enabled)
		assert.Equal(t, []string{"a@example.com", "b@example.com"}, res.EmailAddresses)
		assert.Equal(t, severities("SEVERITY_CRITICAL", "SEVERITY_WARNING"), res.Severities)

		change(t, server_service.ChangeSettingsBody{EnableAdvisorNotifications: new(false)})
		res = get(t)
		assert.False(t, res.Enabled)
		assert.Equal(t, []string{"a@example.com", "b@example.com"}, res.EmailAddresses)
		assert.Equal(t, severities("SEVERITY_CRITICAL", "SEVERITY_WARNING"), res.Severities)
	})

	t.Run("an empty list clears the recipients", func(t *testing.T) {
		change(t, server_service.ChangeSettingsBody{
			EnableAdvisorNotifications: new(false),
			AdvisorNotificationEmailAddresses: &server_service.ChangeSettingsParamsBodyAdvisorNotificationEmailAddresses{
				Values: []string{"a@example.com"},
			},
		})
		require.Equal(t, []string{"a@example.com"}, get(t).EmailAddresses)

		change(t, server_service.ChangeSettingsBody{
			AdvisorNotificationEmailAddresses: &server_service.ChangeSettingsParamsBodyAdvisorNotificationEmailAddresses{
				Values: []string{},
			},
		})
		assert.Empty(t, get(t).EmailAddresses)
	})
}

// quietHour returns an hour, in UTC, at least six hours away from now, so that
// the schedules a test creates for that hour do not fall due while it runs.
func quietHour() int {
	if time.Now().UTC().Hour() < 12 {
		return 18
	}
	return 6
}

// nextOccurrence returns the first time after the given one that falls on one of the days at the time of day, in UTC.
func nextOccurrence(after time.Time, days []int32, timeOfDay string) time.Time {
	clock, err := time.Parse("15:04", timeOfDay)
	if err != nil {
		return time.Time{}
	}

	after = after.UTC()
	start := time.Date(after.Year(), after.Month(), after.Day(), clock.Hour(), clock.Minute(), 0, 0, time.UTC)
	for i := range 8 {
		at := start.AddDate(0, 0, i)
		if at.After(after) && slices.ContainsFunc(days, func(d int32) bool { return int(d) == int(at.Weekday()) }) {
			return at
		}
	}
	return time.Time{}
}

// scheduleName returns a unique schedule name well within the 100-character limit.
func scheduleName(tag string) string {
	return "api-test-schedule-" + tag + "-" + uuid.NewString()
}

// severities converts severity names to the form the API clients take.
func severities(names ...string) []*string {
	res := make([]*string, 0, len(names))
	for _, name := range names {
		res = append(res, new(name))
	}
	return res
}

// createSchedule creates an Advisor schedule and deletes it when the test ends.
func createSchedule(t *testing.T, body advisor.CreateAdvisorScheduleBody) *advisor.CreateAdvisorScheduleOKBodySchedule {
	t.Helper()

	resp, err := advisorClient.Default.AdvisorService.CreateAdvisorSchedule(&advisor.CreateAdvisorScheduleParams{
		Body:    body,
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	id := resp.Payload.Schedule.ScheduleID
	t.Cleanup(func() { deleteScheduleIfExists(id) })

	return resp.Payload.Schedule
}

// deleteScheduleIfExists deletes an Advisor schedule a test may have deleted already.
func deleteScheduleIfExists(id string) {
	_, _ = advisorClient.Default.AdvisorService.DeleteAdvisorSchedule(&advisor.DeleteAdvisorScheduleParams{
		ScheduleID: id,
		Context:    pmmapitests.Context,
	})
}

// deleteAllSchedules deletes every Advisor schedule.
func deleteAllSchedules(t *testing.T) {
	t.Helper()

	for _, s := range listSchedules(t) {
		_, err := advisorClient.Default.AdvisorService.DeleteAdvisorSchedule(&advisor.DeleteAdvisorScheduleParams{
			ScheduleID: s.ScheduleID,
			Context:    pmmapitests.Context,
		})
		require.NoError(t, err)
	}
}

func updateSchedule(t *testing.T, id string, body advisor.UpdateAdvisorScheduleBody) *advisor.UpdateAdvisorScheduleOKBodySchedule {
	t.Helper()

	resp, err := advisorClient.Default.AdvisorService.UpdateAdvisorSchedule(&advisor.UpdateAdvisorScheduleParams{
		ScheduleID: id,
		Body:       body,
		Context:    pmmapitests.Context,
	})
	require.NoError(t, err)

	return resp.Payload.Schedule
}

func listSchedules(t *testing.T) []*advisor.ListAdvisorSchedulesOKBodySchedulesItems0 {
	t.Helper()

	resp, err := advisorClient.Default.AdvisorService.ListAdvisorSchedules(&advisor.ListAdvisorSchedulesParams{Context: pmmapitests.Context})
	require.NoError(t, err)

	return resp.Payload.Schedules
}

func findSchedule(t *testing.T, schedules []*advisor.ListAdvisorSchedulesOKBodySchedulesItems0, id string) *advisor.ListAdvisorSchedulesOKBodySchedulesItems0 {
	t.Helper()

	for _, s := range schedules {
		if s.ScheduleID == id {
			return s
		}
	}
	require.FailNow(t, "schedule isn't listed", id)
	return nil
}

// runSchedule starts a run of the Advisor schedule and returns its ID.
func runSchedule(t *testing.T, id string) string {
	t.Helper()

	resp, err := advisorClient.Default.AdvisorService.RunAdvisorSchedule(&advisor.RunAdvisorScheduleParams{
		ScheduleID: id,
		Context:    pmmapitests.Context,
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Payload.RunID)

	return resp.Payload.RunID
}

func getRun(t *testing.T, id string) *advisor.ListRunsOKBodyResultsItems0 {
	t.Helper()

	resp, err := advisorClient.Default.AdvisorService.ListRuns(&advisor.ListRunsParams{
		RunID:   &id,
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	require.Len(t, resp.Payload.Results, 1)

	return resp.Payload.Results[0]
}

// latestRun returns the number of Advisor runs and the ID of the latest one.
func latestRun(t *testing.T) (int32, string) {
	t.Helper()

	resp, err := advisorClient.Default.AdvisorService.ListRuns(&advisor.ListRunsParams{
		PageSize: new(int32(1)),
		Context:  pmmapitests.Context,
	})
	require.NoError(t, err)

	var latest string
	if len(resp.Payload.Results) != 0 {
		latest = resp.Payload.Results[0].ID
	}
	return resp.Payload.TotalItems, latest
}

// addStallingCheck adds a MySQL service at the plain HTTP port of PMM Server's
// nginx and a custom check that queries it, and returns the check name and the
// service ID. A MySQL client waits for a greeting that nginx never sends, so a
// run of the check stays active until the pmm-agent action times out (15s).
func addStallingCheck(t *testing.T) (string, string) {
	t.Helper()

	resp, err := managementClient.Default.ManagementService.AddService(&mservice.AddServiceParams{
		Body: mservice.AddServiceBody{
			Mysql: &mservice.AddServiceParamsBodyMysql{
				NodeID:              "pmm-server",
				PMMAgentID:          "pmm-server",
				ServiceName:         pmmapitests.TestString(t, "stalling-target"),
				Address:             "127.0.0.1",
				Port:                8080,
				Username:            "username",
				SkipConnectionCheck: true,
			},
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	serviceID := resp.Payload.Mysql.Service.ServiceID
	t.Cleanup(func() { pmmapitests.RemoveServices(t, serviceID) })

	name := customCheckName(t)
	check := customCheckBody(name)
	check.Queries = []*advisor.CreateAdvisorCheckParamsBodyCheckQueriesItems0{{Type: "MYSQL_SELECT", Query: "1"}}
	_, err = advisorClient.Default.AdvisorService.CreateAdvisorCheck(&advisor.CreateAdvisorCheckParams{
		Body:    advisor.CreateAdvisorCheckBody{Check: check},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := advisorClient.Default.AdvisorService.DeleteAdvisorCheck(&advisor.DeleteAdvisorCheckParams{
			Name:    name,
			Context: pmmapitests.Context,
		})
		assert.NoError(t, err)
	})

	return name, serviceID
}

// requestAs sends a request to PMM Server as the Grafana user whose password is
// its login, and returns the response status and body.
func requestAs(t *testing.T, login, method, path string, body any) (int, []byte) {
	t.Helper()

	u, err := url.Parse(pmmapitests.BaseURL.String())
	require.NoError(t, err)
	u.User = url.UserPassword(login, login)
	u.Path = path

	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(pmmapitests.Context, method, u.String(), r)
	require.NoError(t, err)

	resp, b := doRequest(t, pmmapitests.HTTPClient, req) //nolint:bodyclose
	return resp.StatusCode, b
}

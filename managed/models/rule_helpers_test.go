// Copyright (C) 2024 Percona LLC
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

package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/percona-platform/saas/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
)

func TestRules(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	nonEmptyFilters := []models.Filter{{Type: models.Equal, Key: "value", Val: "10"}}

	t.Run("create", func(t *testing.T) {
		t.Run("normal", func(t *testing.T) {
			tx, err := db.Begin()
			require.NoError(t, err)
			defer func() {
				require.NoError(t, tx.Rollback())
			}()

			q := tx.Querier

			template := createTemplate(t, q)
			channel := createChannel(t, q)

			params := createCreateRuleParams(t, template, channel.ID, nonEmptyFilters)
			rule, err := models.CreateRule(q, params)
			require.NoError(t, err)

			assert.NotEmpty(t, rule.ID)
			assert.Equal(t, template.Name, rule.TemplateName)
			assert.Equal(t, template.Summary, rule.Summary)
			assert.Equal(t, template.Severity, rule.DefaultSeverity)
			assert.Equal(t, template.For, rule.DefaultFor)
			assert.Equal(t, params.Name, rule.Name)
			assert.Equal(t, params.Disabled, rule.Disabled)
			assert.Equal(t, params.ParamsValues, rule.ParamsValues)
			assert.Equal(t, params.For, rule.For)
			assert.Equal(t, models.Severity(common.Warning), rule.Severity)

			customLabels, err := rule.GetCustomLabels()
			require.NoError(t, err)
			assert.Equal(t, params.CustomLabels, customLabels)

			labels, err := rule.GetLabels()
			require.NoError(t, err)
			templateLabels, err := template.GetLabels()
			require.NoError(t, err)
			assert.Equal(t, templateLabels, labels)

			annotations, err := rule.GetAnnotations()
			require.NoError(t, err)
			templateAnnotations, err := template.GetAnnotations()
			require.NoError(t, err)
			assert.Equal(t, templateAnnotations, annotations)

			assert.Equal(t, params.Filters, rule.Filters)
			assert.ElementsMatch(t, params.ChannelIDs, rule.ChannelIDs)
		})

		t.Run("rule without channel and filters", func(t *testing.T) {
			tx, err := db.Begin()
			require.NoError(t, err)
			defer func() {
				require.NoError(t, tx.Rollback())
			}()

			q := tx.Querier

			template := createTemplate(t, q)

			params := createCreateRuleParams(t, template, "", nil)
			rule, err := models.CreateRule(q, params)
			require.NoError(t, err)

			assert.NotEmpty(t, rule.ID)
			assert.Equal(t, template.Name, rule.TemplateName)
			assert.Equal(t, template.Summary, rule.Summary)
			assert.Equal(t, template.Severity, rule.DefaultSeverity)
			assert.Equal(t, template.For, rule.DefaultFor)
			assert.Equal(t, params.Name, rule.Name)
			assert.Equal(t, params.Disabled, rule.Disabled)
			assert.Equal(t, params.ParamsValues, rule.ParamsValues)
			assert.Equal(t, params.For, rule.For)
			assert.Equal(t, models.Severity(common.Warning), rule.Severity)

			customLabels, err := rule.GetCustomLabels()
			require.NoError(t, err)
			assert.Equal(t, params.CustomLabels, customLabels)

			labels, err := rule.GetLabels()
			require.NoError(t, err)
			templateLabels, err := template.GetLabels()
			require.NoError(t, err)
			assert.Equal(t, templateLabels, labels)

			annotations, err := rule.GetAnnotations()
			require.NoError(t, err)
			templateAnnotations, err := template.GetAnnotations()
			require.NoError(t, err)
			assert.Equal(t, templateAnnotations, annotations)

			assert.Nil(t, rule.Filters)
			assert.Empty(t, rule.ChannelIDs)
		})

		t.Run("unknown channel", func(t *testing.T) {
			tx, err := db.Begin()
			require.NoError(t, err)
			defer func() {
				require.NoError(t, tx.Rollback())
			}()

			q := tx.Querier

			template := createTemplate(t, q)
			channelID := uuid.New().String()

			params := createCreateRuleParams(t, template, channelID, nonEmptyFilters)
			_, err = models.CreateRule(q, params)
			tests.AssertGRPCError(t, status.Newf(codes.NotFound, "Failed to find all required channels: %v.", []string{channelID}), err)
		})
	})

	t.Run("change", func(t *testing.T) {
		t.Run("normal", func(t *testing.T) {
			tx, err := db.Begin()
			require.NoError(t, err)
			defer func() {
				require.NoError(t, tx.Rollback())
			}()

			q := tx.Querier

			template := createTemplate(t, q)
			channel := createChannel(t, q)
			rule, err := models.CreateRule(q, createCreateRuleParams(t, template, channel.ID, nonEmptyFilters))
			require.NoError(t, err)

			newChannel := createChannel(t, q)

			params := &models.ChangeRuleParams{
				Name:         "summary",
				Disabled:     false,
				ParamsValues: nil,
				For:          3 * time.Second,
				Severity:     models.Severity(common.Info),
				CustomLabels: map[string]string{"test": "example"},
				Filters:      []models.Filter{{Type: models.Equal, Key: "number", Val: "42"}},
				ChannelIDs:   []string{newChannel.ID},
			}

			updated, err := models.ChangeRule(q, rule.ID, params)
			require.NoError(t, err)

			assert.NotEmpty(t, rule.ID, updated.ID)
			assert.Equal(t, template.Name, updated.TemplateName)
			assert.Equal(t, template.Summary, updated.Summary)
			assert.Equal(t, template.Severity, updated.DefaultSeverity)
			assert.Equal(t, template.For, updated.DefaultFor)
			assert.Equal(t, params.Name, updated.Name)
			assert.Equal(t, params.Disabled, updated.Disabled)
			assert.Equal(t, params.ParamsValues, updated.ParamsValues)
			assert.Equal(t, params.For, updated.For)
			assert.Equal(t, models.Severity(common.Info), updated.Severity)

			labels, err := updated.GetCustomLabels()
			require.NoError(t, err)
			assert.Equal(t, params.CustomLabels, labels)
			assert.Equal(t, params.Filters, updated.Filters)
			assert.ElementsMatch(t, params.ChannelIDs, updated.ChannelIDs)
		})

		t.Run("unknown channel", func(t *testing.T) {
			tx, err := db.Begin()
			require.NoError(t, err)
			defer func() {
				require.NoError(t, tx.Rollback())
			}()

			q := tx.Querier

			template := createTemplate(t, q)
			channel := createChannel(t, q)
			rule, err := models.CreateRule(q, createCreateRuleParams(t, template, channel.ID, nonEmptyFilters))
			require.NoError(t, err)

			newChannelID := uuid.New().String()

			params := &models.ChangeRuleParams{
				Disabled:     false,
				ParamsValues: nil,
				For:          3 * time.Second,
				Severity:     models.Severity(common.Info),
				CustomLabels: map[string]string{"test": "example"},
				Filters:      []models.Filter{{Type: models.Equal, Key: "number", Val: "42"}},
				ChannelIDs:   []string{newChannelID},
			}

			_, err = models.ChangeRule(q, rule.ID, params)
			tests.AssertGRPCError(t, status.Newf(codes.NotFound, "Failed to find all required channels: %v.", []string{newChannelID}), err)
		})
	})

	t.Run("remove", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()

		q := tx.Querier

		template := createTemplate(t, q)
		channel := createChannel(t, q)

		params := createCreateRuleParams(t, template, channel.ID, nonEmptyFilters)
		rule, err := models.CreateRule(q, params)
		require.NoError(t, err)

		err = models.RemoveRule(q, rule.ID)
		require.NoError(t, err)

		rules, err := models.FindRules(q)
		require.NoError(t, err)
		assert.Empty(t, rules)
	})

	t.Run("list", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()

		q := tx.Querier

		template := createTemplate(t, q)
		channel := createChannel(t, q)

		params := createCreateRuleParams(t, template, channel.ID, nonEmptyFilters)
		rule, err := models.CreateRule(q, params)
		require.NoError(t, err)

		rules, err := models.FindRules(q)
		require.NoError(t, err)
		assert.Len(t, rules, 1)

		actual := rules[0]
		assert.NotEmpty(t, rule.ID)
		assert.Equal(t, rule.Summary, actual.Summary)
		assert.Equal(t, rule.TemplateName, actual.TemplateName)
		assert.Equal(t, rule.Disabled, actual.Disabled)
		assert.Equal(t, rule.ParamsValues, actual.ParamsValues)
		assert.Equal(t, rule.For, actual.For)
		assert.Equal(t, rule.Severity, actual.Severity)
		assert.Equal(t, rule.CustomLabels, actual.CustomLabels)
		assert.Equal(t, rule.Filters, actual.Filters)
		assert.ElementsMatch(t, rule.ChannelIDs, actual.ChannelIDs)
	})
}

func createCreateRuleParams(t *testing.T, template *models.Template, channelID string, filters []models.Filter) *models.CreateRuleParams {
	t.Helper()

	labels, err := template.GetLabels()
	require.NoError(t, err)

	annotations, err := template.GetAnnotations()
	require.NoError(t, err)

	rule := &models.CreateRuleParams{
		TemplateName: template.Name,
		Name:         "rule name",
		Summary:      template.Summary,
		Disabled:     true,
		ParamsValues: []models.AlertExprParamValue{
			{
				Name:       "test",
				Type:       models.Float,
				FloatValue: 3.14,
			},
		},
		DefaultFor:      template.For,
		For:             5 * time.Second,
		DefaultSeverity: template.Severity,
		Severity:        models.Severity(common.Warning),
		CustomLabels:    map[string]string{"foo": "bar"},
		Labels:          labels,
		Annotations:     annotations,
	}
	if channelID != "" {
		rule.ChannelIDs = []string{channelID}
	}

	if filters != nil {
		rule.Filters = filters
	}

	return rule
}

func createTemplate(t *testing.T, q *reform.Querier) *models.Template {
	t.Helper()

	template, err := models.CreateTemplate(q, createTemplateParams(uuid.New().String()))
	require.NoError(t, err)
	return template
}

func createChannel(t *testing.T, q *reform.Querier) *models.Channel {
	t.Helper()

	params := models.CreateChannelParams{
		Summary: "some summary",
		EmailConfig: &models.EmailConfig{
			To: []string{"test@test.test"},
		},
		Disabled: false,
	}

	channel, err := models.CreateChannel(q, &params)
	require.NoError(t, err)

	return channel
}

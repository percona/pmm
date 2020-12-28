// pmm-managed
// Copyright (C) 2017 Percona LLC
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

	"github.com/brianvoe/gofakeit"
	"github.com/percona-platform/saas/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/testdb"
	"github.com/percona/pmm-managed/utils/tests"
)

func TestRules(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	t.Run("create", func(t *testing.T) {
		t.Run("normal", func(t *testing.T) {
			tx, err := db.Begin()
			require.NoError(t, err)
			defer func() {
				require.NoError(t, tx.Rollback())
			}()

			q := tx.Querier

			templateName := createTemplate(t, q)
			channelID := createChannel(t, q)

			params := createCreateRuleParams(templateName, channelID)
			rule, err := models.CreateRule(q, params)
			require.NoError(t, err)

			assert.NotEmpty(t, rule.ID)
			assert.Equal(t, templateName, rule.TemplateName)
			assert.Equal(t, params.Summary, rule.Summary)
			assert.Equal(t, params.Disabled, rule.Disabled)
			assert.Equal(t, params.RuleParams, rule.Params)
			assert.Equal(t, params.For, rule.For)
			assert.Equal(t, models.Severity(common.Warning), rule.Severity)

			labels, err := rule.GetCustomLabels()
			require.NoError(t, err)
			assert.Equal(t, params.CustomLabels, labels)
			assert.Equal(t, params.Filters, rule.Filters)
			assert.ElementsMatch(t, params.ChannelIDs, rule.ChannelIDs)
		})

		t.Run("unknown channel", func(t *testing.T) {
			tx, err := db.Begin()
			require.NoError(t, err)
			defer func() {
				require.NoError(t, tx.Rollback())
			}()

			q := tx.Querier

			templateName := createTemplate(t, q)
			channelID := gofakeit.UUID()

			params := createCreateRuleParams(templateName, channelID)
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

			templateName := createTemplate(t, q)
			channelID := createChannel(t, q)
			rule, err := models.CreateRule(q, createCreateRuleParams(templateName, channelID))
			require.NoError(t, err)

			newChannelID := createChannel(t, q)

			params := &models.ChangeRuleParams{
				Disabled:     false,
				RuleParams:   nil,
				For:          3 * time.Second,
				Severity:     models.Severity(common.Info),
				CustomLabels: map[string]string{"test": "example"},
				Filters:      []models.Filter{{Type: models.Equal, Key: "number", Val: "42"}},
				ChannelIDs:   []string{newChannelID},
			}

			updated, err := models.ChangeRule(q, rule.ID, params)
			require.NoError(t, err)

			assert.NotEmpty(t, rule.ID, updated.ID)
			assert.Equal(t, templateName, updated.TemplateName)
			assert.Equal(t, params.Disabled, updated.Disabled)
			assert.Equal(t, params.RuleParams, updated.Params)
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

			templateName := createTemplate(t, q)
			channelID := createChannel(t, q)
			rule, err := models.CreateRule(q, createCreateRuleParams(templateName, channelID))
			require.NoError(t, err)

			newChannelID := gofakeit.UUID()

			params := &models.ChangeRuleParams{
				Disabled:     false,
				RuleParams:   nil,
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

		templateName := createTemplate(t, q)
		channelID := createChannel(t, q)

		params := createCreateRuleParams(templateName, channelID)
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

		templateName := createTemplate(t, q)
		channelID := createChannel(t, q)

		params := createCreateRuleParams(templateName, channelID)
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
		assert.Equal(t, rule.Params, actual.Params)
		assert.Equal(t, rule.For, actual.For)
		assert.Equal(t, rule.Severity, actual.Severity)
		assert.Equal(t, rule.CustomLabels, actual.CustomLabels)
		assert.Equal(t, rule.Filters, actual.Filters)
		assert.ElementsMatch(t, rule.ChannelIDs, actual.ChannelIDs)
	})
}

func createCreateRuleParams(templateName, channelID string) *models.CreateRuleParams {
	return &models.CreateRuleParams{
		TemplateName: templateName,
		Disabled:     true,
		RuleParams: []models.RuleParam{
			{
				Name:       "test",
				Type:       models.Float,
				FloatValue: 3.14,
			},
		},
		For:          5 * time.Second,
		Severity:     models.Severity(common.Warning),
		CustomLabels: map[string]string{"foo": "bar"},
		Filters:      []models.Filter{{Type: models.Equal, Key: "value", Val: "10"}},
		ChannelIDs:   []string{channelID},
	}
}

func createTemplate(t *testing.T, q *reform.Querier) string {
	templateName := gofakeit.UUID()
	_, err := models.CreateTemplate(q, createTemplateParams(templateName))
	require.NoError(t, err)
	return templateName
}

func createChannel(t *testing.T, q *reform.Querier) string {
	params := models.CreateChannelParams{
		Summary: "some summary",
		EmailConfig: &models.EmailConfig{
			To: []string{"test@test.test"},
		},
		Disabled: false,
	}

	expected, err := models.CreateChannel(q, &params)
	require.NoError(t, err)

	return expected.ID
}

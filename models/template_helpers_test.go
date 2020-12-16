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

	"github.com/brianvoe/gofakeit"
	"github.com/percona-platform/saas/pkg/alert"
	"github.com/percona-platform/saas/pkg/common"
	"github.com/percona/promconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/testdb"
)

func TestRuleTemplates(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	t.Run("create", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()

		q := tx.Querier

		params := createTemplateParams(gofakeit.UUID())

		created, err := models.CreateTemplate(q, params)
		require.NoError(t, err)

		assert.Equal(t, params.Template.Name, created.Name)
		assert.Equal(t, params.Template.Version, created.Version)
		assert.Equal(t, params.Template.Summary, created.Summary)
		assert.ElementsMatch(t, params.Template.Tiers, created.Tiers)
		assert.Equal(t, params.Template.Expr, created.Expr)
		assert.Equal(t,
			models.TemplateParams{{
				Name:    params.Template.Params[0].Name,
				Summary: params.Template.Params[0].Summary,
				Unit:    params.Template.Params[0].Unit,
				Type:    models.Float,
				FloatParam: &models.FloatParam{
					Default: params.Template.Params[0].Value.(float64),
					Min:     params.Template.Params[0].Range[0].(float64),
					Max:     params.Template.Params[0].Range[1].(float64),
				},
			}},
			created.Params)
		assert.EqualValues(t, params.Template.For, created.For)
		assert.Equal(t, models.WarningSeverity, created.Severity)

		labels, err := created.GetLabels()
		require.NoError(t, err)
		assert.Equal(t, params.Template.Labels, labels)

		annotations, err := created.GetAnnotations()
		require.NoError(t, err)
		assert.Equal(t, params.Template.Annotations, annotations)

		assert.Equal(t, params.Source, created.Source)
	})

	t.Run("change", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()

		q := tx.Querier

		name := gofakeit.UUID()

		cParams := createTemplateParams(name)
		_, err = models.CreateTemplate(q, cParams)
		require.NoError(t, err)

		uParams := changeTemplateParams(name)
		updated, err := models.ChangeTemplate(q, uParams)
		require.NoError(t, err)

		assert.Equal(t, uParams.Template.Name, updated.Name)
		assert.Equal(t, uParams.Template.Version, updated.Version)
		assert.Equal(t, uParams.Template.Summary, updated.Summary)
		assert.ElementsMatch(t, uParams.Template.Tiers, updated.Tiers)
		assert.Equal(t, uParams.Template.Expr, updated.Expr)
		assert.Equal(t,
			models.TemplateParams{{
				Name:    uParams.Template.Params[0].Name,
				Summary: uParams.Template.Params[0].Summary,
				Unit:    uParams.Template.Params[0].Unit,
				Type:    models.Float,
				FloatParam: &models.FloatParam{
					Default: uParams.Template.Params[0].Value.(float64),
					Min:     uParams.Template.Params[0].Range[0].(float64),
					Max:     uParams.Template.Params[0].Range[1].(float64),
				},
			}},
			updated.Params)
		assert.EqualValues(t, uParams.Template.For, updated.For)
		assert.Equal(t, models.WarningSeverity, updated.Severity)

		labels, err := updated.GetLabels()
		require.NoError(t, err)
		assert.Equal(t, uParams.Template.Labels, labels)

		annotations, err := updated.GetAnnotations()
		require.NoError(t, err)
		assert.Equal(t, uParams.Template.Annotations, annotations)

		assert.Equal(t, cParams.Source, updated.Source)
	})

	t.Run("remove", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()

		q := tx.Querier

		name := gofakeit.UUID()

		_, err = models.CreateTemplate(q, createTemplateParams(name))
		require.NoError(t, err)

		err = models.RemoveTemplate(q, name)
		require.NoError(t, err)

		templates, err := models.FindTemplates(q)
		require.NoError(t, err)

		assert.Empty(t, templates)
	})

	t.Run("list", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()

		q := tx.Querier

		created, err := models.CreateTemplate(q, createTemplateParams(gofakeit.UUID()))
		require.NoError(t, err)

		templates, err := models.FindTemplates(q)
		require.NoError(t, err)
		assert.Len(t, templates, 1)

		actual := templates[0]

		assert.Equal(t, created.Name, actual.Name)
		assert.Equal(t, created.Version, actual.Version)
		assert.Equal(t, created.Summary, actual.Summary)
		assert.ElementsMatch(t, created.Tiers, actual.Tiers)
		assert.Equal(t, created.Expr, actual.Expr)
		assert.Equal(t, created.Params, actual.Params)
		assert.EqualValues(t, created.For, actual.For)
		assert.Equal(t, created.Severity, actual.Severity)
		assert.Equal(t, created.Labels, actual.Labels)
		assert.Empty(t, actual.Annotations)
		assert.Equal(t, created.Source, actual.Source)
	})
}

func createTemplateParams(name string) *models.CreateTemplateParams {
	return &models.CreateTemplateParams{
		Template: &alert.Template{
			Name:    name,
			Version: 1,
			Summary: gofakeit.Quote(),
			Tiers:   []common.Tier{common.Anonymous},
			Expr:    gofakeit.Quote(),
			Params: []alert.Parameter{{
				Name:    gofakeit.UUID(),
				Summary: gofakeit.Quote(),
				Unit:    gofakeit.Letter(),
				Type:    alert.Float,
				Range:   []interface{}{float64(10), float64(100)},
				Value:   float64(50),
			}},
			For:         3,
			Severity:    common.Warning,
			Labels:      map[string]string{"foo": "bar"},
			Annotations: nil,
		},
		Source: "USER_FILE",
	}
}

func changeTemplateParams(name string) *models.ChangeTemplateParams {
	return &models.ChangeTemplateParams{
		Template: &alert.Template{
			Name:    name,
			Version: 1,
			Summary: gofakeit.Quote(),
			Tiers:   []common.Tier{common.Anonymous},
			Expr:    gofakeit.Quote(),
			Params: []alert.Parameter{{
				Name:    gofakeit.UUID(),
				Summary: gofakeit.Quote(),
				Unit:    gofakeit.Letter(),
				Type:    alert.Float,
				Range:   []interface{}{float64(10), float64(100)},
				Value:   float64(50),
			}},
			For:         promconfig.Duration(gofakeit.Number(1, 100)),
			Severity:    common.Warning,
			Labels:      map[string]string{"foo": "bar"},
			Annotations: nil,
		},
	}
}

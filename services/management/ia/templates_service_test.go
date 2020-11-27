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

package ia

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/testdb"
)

const (
	testBadTemplates     = "../../../testdata/ia/bad/*.yml"
	testBuiltinTemplates = "../../../testdata/ia/builtin/*.yml"
	testUser2Templates   = "../../../testdata/ia/user2/*.yml"
	testUserTemplates    = "../../../testdata/ia/user/*.yml"
	testMissingTemplates = "/no/such/path/*.yml"
)

func TestCollect(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	t.Run("bad and missing template paths", func(t *testing.T) {
		t.Parallel()

		svc := NewTemplatesService(db)
		svc.builtinTemplatesPath = testMissingTemplates
		svc.userTemplatesPath = testBadTemplates
		svc.collect(ctx)

		require.Empty(t, svc.getCollected(ctx))
	})

	t.Run("valid template paths", func(t *testing.T) {
		t.Parallel()

		svc := NewTemplatesService(db)
		svc.builtinTemplatesPath = testBuiltinTemplates
		svc.userTemplatesPath = testUserTemplates
		svc.collect(ctx)

		rules := svc.getCollected(ctx)
		require.NotEmpty(t, rules)
		require.Len(t, rules, 2)
		assert.Contains(t, rules, "builtin_rule")
		assert.Contains(t, rules, "user_rule")

		// check whether map was cleared and updated on a subsequent call
		svc.userTemplatesPath = testUser2Templates
		svc.collect(ctx)

		rules = svc.getCollected(ctx)
		require.NotEmpty(t, rules)
		require.Len(t, rules, 2)
		assert.NotContains(t, rules, "user_rule")
		assert.Contains(t, rules, "builtin_rule")
		assert.Contains(t, rules, "user2_rule")
	})
}

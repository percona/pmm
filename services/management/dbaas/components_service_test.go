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

package dbaas

import (
	"context"
	"testing"
	"time"

	goversion "github.com/hashicorp/go-version"
	dbaasv1beta1 "github.com/percona/pmm/api/managementpb/dbaas"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComponentService(t *testing.T) {
	t.Run("MatrixConversion", func(t *testing.T) {
		cs := &componentsService{}

		input := map[string]component{
			"5.7.26-31.37":   {ImagePath: "percona/percona-xtradb-cluster:5.7.26-31.37", ImageHash: "9d43d8e435e4aca5c694f726cc736667cb938158635c5f01a0e9412905f1327f", Status: "available", Critical: false},
			"5.7.27-31.39":   {ImagePath: "percona/percona-xtradb-cluster:5.7.27-31.39", ImageHash: "7d8eb4d2031c32c6e96451655f359d8e5e8e047dc95bada9a28c41c158876c26", Status: "available", Critical: false},
			"5.7.28-31.41.2": {ImagePath: "percona/percona-xtradb-cluster:5.7.28-31.41.2", ImageHash: "fccd6525aaeedb5e436e9534e2a63aebcf743c043526dd05dba8519ebddc8b30", Status: "available", Critical: true},
			"5.7.29-31.43":   {ImagePath: "percona/percona-xtradb-cluster:5.7.29-31.43", ImageHash: "85fb479de073770280ae601cf3ec22dc5c8cca4c8b0dc893b09503767338e6f9", Status: "available", Critical: false},
			"5.7.30-31.43":   {ImagePath: "percona/percona-xtradb-cluster:5.7.30-31.43", ImageHash: "b03a060e9261b37288a2153c78f86dcfc53367c36e1bcdcae046dd2d0b0721af", Status: "available", Critical: false},
			"5.7.31-31.45":   {ImagePath: "percona/percona-xtradb-cluster:5.7.31-31.45", ImageHash: "3852cef43cc0c6aa791463ba6279e59dcdac3a4fb1a5616c745c1b3c68041dc2", Status: "available", Critical: false},
			"5.7.31-31.45.2": {ImagePath: "percona/percona-xtradb-cluster:5.7.31-31.45.2", ImageHash: "0decf85c7c7afacc438f5fe355dc8320ea7ffc7018ca2cb6bda3ac0c526ae172", Status: "available", Critical: false},
			"5.7.32-31.47":   {ImagePath: "percona/percona-xtradb-cluster:5.7.32-31.47", ImageHash: "7b095019ad354c336494248d6080685022e2ed46e3b53fc103b25cd12c95952b", Status: "recommended", Critical: false},
			"8.0.19-10.1":    {ImagePath: "percona/percona-xtradb-cluster:8.0.19-10.1", ImageHash: "1058ae8eded735ebdf664807aad7187942fc9a1170b3fd0369574cb61206b63a", Status: "available", Critical: false},
			"8.0.20-11.1":    {ImagePath: "percona/percona-xtradb-cluster:8.0.20-11.1", ImageHash: "54b1b2f5153b78b05d651034d4603a13e685cbb9b45bfa09a39864fa3f169349", Status: "available", Critical: false},
			"8.0.20-11.2":    {ImagePath: "percona/percona-xtradb-cluster:8.0.20-11.2", ImageHash: "feda5612db18da824e971891d6084465aa9cdc9918c18001cd95ba30916da78b", Status: "available", Critical: false},
			"8.0.21-12.1":    {ImagePath: "percona/percona-xtradb-cluster:8.0.21-12.1", ImageHash: "d95cf39a58f09759408a00b519fe0d0b19c1b28332ece94349dd5e9cdbda017e", Status: "recommended", Critical: false},
		}
		m := cs.matrix(input, nil)

		expected := map[string]*dbaasv1beta1.Component{
			"5.7.26-31.37":   {ImagePath: "percona/percona-xtradb-cluster:5.7.26-31.37", ImageHash: "9d43d8e435e4aca5c694f726cc736667cb938158635c5f01a0e9412905f1327f", Status: "available", Critical: false},
			"5.7.27-31.39":   {ImagePath: "percona/percona-xtradb-cluster:5.7.27-31.39", ImageHash: "7d8eb4d2031c32c6e96451655f359d8e5e8e047dc95bada9a28c41c158876c26", Status: "available", Critical: false},
			"5.7.28-31.41.2": {ImagePath: "percona/percona-xtradb-cluster:5.7.28-31.41.2", ImageHash: "fccd6525aaeedb5e436e9534e2a63aebcf743c043526dd05dba8519ebddc8b30", Status: "available", Critical: true},
			"5.7.29-31.43":   {ImagePath: "percona/percona-xtradb-cluster:5.7.29-31.43", ImageHash: "85fb479de073770280ae601cf3ec22dc5c8cca4c8b0dc893b09503767338e6f9", Status: "available", Critical: false},
			"5.7.30-31.43":   {ImagePath: "percona/percona-xtradb-cluster:5.7.30-31.43", ImageHash: "b03a060e9261b37288a2153c78f86dcfc53367c36e1bcdcae046dd2d0b0721af", Status: "available", Critical: false},
			"5.7.31-31.45":   {ImagePath: "percona/percona-xtradb-cluster:5.7.31-31.45", ImageHash: "3852cef43cc0c6aa791463ba6279e59dcdac3a4fb1a5616c745c1b3c68041dc2", Status: "available", Critical: false},
			"5.7.31-31.45.2": {ImagePath: "percona/percona-xtradb-cluster:5.7.31-31.45.2", ImageHash: "0decf85c7c7afacc438f5fe355dc8320ea7ffc7018ca2cb6bda3ac0c526ae172", Status: "available", Critical: false},
			"5.7.32-31.47":   {ImagePath: "percona/percona-xtradb-cluster:5.7.32-31.47", ImageHash: "7b095019ad354c336494248d6080685022e2ed46e3b53fc103b25cd12c95952b", Status: "recommended", Critical: false},
			"8.0.19-10.1":    {ImagePath: "percona/percona-xtradb-cluster:8.0.19-10.1", ImageHash: "1058ae8eded735ebdf664807aad7187942fc9a1170b3fd0369574cb61206b63a", Status: "available", Critical: false},
			"8.0.20-11.1":    {ImagePath: "percona/percona-xtradb-cluster:8.0.20-11.1", ImageHash: "54b1b2f5153b78b05d651034d4603a13e685cbb9b45bfa09a39864fa3f169349", Status: "available", Critical: false},
			"8.0.20-11.2":    {ImagePath: "percona/percona-xtradb-cluster:8.0.20-11.2", ImageHash: "feda5612db18da824e971891d6084465aa9cdc9918c18001cd95ba30916da78b", Status: "available", Critical: false},
			"8.0.21-12.1":    {ImagePath: "percona/percona-xtradb-cluster:8.0.21-12.1", ImageHash: "d95cf39a58f09759408a00b519fe0d0b19c1b28332ece94349dd5e9cdbda017e", Status: "recommended", Critical: false, Default: true},
		}

		assert.Equal(t, expected, m)
	})

	t.Run("EmptyMatrix", func(t *testing.T) {
		cs := &componentsService{}
		m := cs.matrix(map[string]component{}, nil)
		assert.Equal(t, map[string]*dbaasv1beta1.Component{}, m)
	})
}

func TestFilteringOutOfDisabledVersions(t *testing.T) {
	t.Parallel()
	c := &componentsService{
		l:                    logrus.WithField("component", "components_service"),
		db:                   nil,
		dbaasClient:          dbaasClient(nil),
		versionServiceClient: NewVersionServiceClient("https://check.percona.com/versions/v1"),
	}

	t.Run("mongod", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()

		params := componentsParams{
			operator:        psmdbOperator,
			operatorVersion: "1.6.0",
		}
		versions, err := c.versions(ctx, params)
		require.NoError(t, err)
		parsedDisabledVersion, err := goversion.NewVersion("4.2.0")
		require.NoError(t, err)
		for _, v := range versions {
			for version := range v.Matrix.Mongod {
				parsedVersion, err := goversion.NewVersion(version)
				require.NoError(t, err)
				assert.Truef(t, parsedVersion.GreaterThanOrEqual(parsedDisabledVersion), "%s is not greater or equal to 4.2.0", version)
			}
		}
	})

	t.Run("pxc", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()

		params := componentsParams{
			operator:        pxcOperator,
			operatorVersion: "1.7.0",
		}
		versions, err := c.versions(ctx, params)
		require.NoError(t, err)
		parsedDisabledVersion, err := goversion.NewVersion("8.0.0")
		require.NoError(t, err)
		for _, v := range versions {
			for version := range v.Matrix.Pxc {
				parsedVersion, err := goversion.NewVersion(version)
				require.NoError(t, err)
				assert.True(t, parsedVersion.GreaterThanOrEqual(parsedDisabledVersion), "%s is not greater or equal to 8.0.0", version)
			}
		}
	})
}

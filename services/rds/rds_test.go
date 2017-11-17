// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

// Package rds contains business logic of working with AWS RDS.
package rds

import (
	"context"
	"reflect"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/mysql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/logger"
	"github.com/percona/pmm-managed/utils/tests"
)

func TestDiscover(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		ctx, _ := logger.Set(context.Background(), t.Name())
		accessKey, secretKey := tests.GetAWSKeys(t)
		db := tests.OpenTestDB(t)
		defer db.Close()
		svc, err := NewService(reform.NewDB(db, mysql.Dialect, reform.NewPrintfLogger(t.Logf)))
		require.NoError(t, err)

		actual, err := svc.Discover(ctx, accessKey, secretKey)
		require.NoError(t, err)
		expected := []Instance{{
			Node: models.RDSNode{
				Type:   "rds",
				Name:   "mysql57",
				Region: "eu-west-1",
			},
			Service: models.RDSService{
				Type:          "rds",
				Address:       pointer.ToString("mysql57.ckpwzom1xccn.eu-west-1.rds.amazonaws.com"),
				Port:          pointer.ToUint16(3306),
				Engine:        pointer.ToString("mysql"),
				EngineVersion: pointer.ToString("5.7.19"),
			},
		}, {
			Node: models.RDSNode{
				Type:   "rds",
				Name:   "aurora1",
				Region: "us-east-1",
			},
			Service: models.RDSService{
				Type:          "rds",
				Address:       pointer.ToString("aurora1.cdy17lilqrl7.us-east-1.rds.amazonaws.com"),
				Port:          pointer.ToUint16(3306),
				Engine:        pointer.ToString("aurora"),
				EngineVersion: pointer.ToString("5.6.10a"),
			},
		}, {
			Node: models.RDSNode{
				Type:   "rds",
				Name:   "aurora1-us-east-1c",
				Region: "us-east-1",
			},
			Service: models.RDSService{
				Type:          "rds",
				Address:       pointer.ToString("aurora1-us-east-1c.cdy17lilqrl7.us-east-1.rds.amazonaws.com"),
				Port:          pointer.ToUint16(3306),
				Engine:        pointer.ToString("aurora"),
				EngineVersion: pointer.ToString("5.6.10a"),
			},
		}, {
			Node: models.RDSNode{
				Type:   "rds",
				Name:   "mysql56",
				Region: "us-east-1",
			},
			Service: models.RDSService{
				Type:          "rds",
				Address:       pointer.ToString("mysql56.cdy17lilqrl7.us-east-1.rds.amazonaws.com"),
				Port:          pointer.ToUint16(3306),
				Engine:        pointer.ToString("mysql"),
				EngineVersion: pointer.ToString("5.6.35"),
			},
		}}

		// TODO out list is not fixed yet, so check that we receive all expected instances (and maybe something else)
		// assert.Equal(t, expected, actual)
		for _, a := range actual {
			for i, e := range expected {
				if reflect.DeepEqual(a, e) {
					expected = append(expected[:i], expected[i+1:]...)
					break
				}
			}
		}
		assert.Empty(t, expected)
	})

	t.Run("WrongKeys", func(t *testing.T) {
		ctx, _ := logger.Set(context.Background(), t.Name())
		accessKey, secretKey := "AKIAIOSFODNN7EXAMPLE", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
		db := tests.OpenTestDB(t)
		defer db.Close()
		svc, err := NewService(reform.NewDB(db, mysql.Dialect, reform.NewPrintfLogger(t.Logf)))
		require.NoError(t, err)

		res, err := svc.Discover(ctx, accessKey, secretKey)
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `The security token included in the request is invalid.`), err)
		assert.Empty(t, res)
	})
}

func TestAddListRemove(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		ctx, _ := logger.Set(context.Background(), t.Name())
		accessKey, secretKey := tests.GetAWSKeys(t)
		db := tests.OpenTestDB(t)
		defer db.Close()
		svc, err := NewService(reform.NewDB(db, mysql.Dialect, reform.NewPrintfLogger(t.Logf)))
		require.NoError(t, err)

		actual, err := svc.List(ctx)
		require.NoError(t, err)
		assert.Empty(t, actual)

		err = svc.Add(ctx, accessKey, secretKey, &InstanceID{}, "username", "password")
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `RDS instance name is not given.`), err)

		err = svc.Add(ctx, accessKey, secretKey, &InstanceID{"eu-west-1", "mysql57"}, "username", "password")
		assert.NoError(t, err)

		err = svc.Add(ctx, accessKey, secretKey, &InstanceID{"eu-west-1", "mysql57"}, "username", "password")
		tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `RDS instance "mysql57" already exists in region "eu-west-1".`), err)

		actual, err = svc.List(ctx)
		require.NoError(t, err)
		expected := []Instance{{
			Node: models.RDSNode{
				ID:     2,
				Type:   "rds",
				Name:   "mysql57",
				Region: "eu-west-1",
			},
			Service: models.RDSService{
				ID:            1,
				Type:          "rds",
				NodeID:        2,
				AWSAccessKey:  &accessKey,
				AWSSecretKey:  &secretKey,
				Address:       pointer.ToString("mysql57.ckpwzom1xccn.eu-west-1.rds.amazonaws.com"),
				Port:          pointer.ToUint16(3306),
				Engine:        pointer.ToString("mysql"),
				EngineVersion: pointer.ToString("5.7.19"),
			},
		}}
		assert.Equal(t, expected, actual)

		err = svc.Remove(ctx, &InstanceID{})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `RDS instance name is not given.`), err)

		err = svc.Remove(ctx, &InstanceID{"eu-west-1", "mysql57"})
		assert.NoError(t, err)

		err = svc.Remove(ctx, &InstanceID{"eu-west-1", "mysql57"})
		tests.AssertGRPCError(t, status.New(codes.NotFound, `RDS instance "mysql57" not found in region "eu-west-1".`), err)

		actual, err = svc.List(ctx)
		require.NoError(t, err)
		assert.Empty(t, actual)
	})
}

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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-managed/utils/logger"
	"github.com/percona/pmm-managed/utils/tests"
)

func TestDiscover(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		ctx, _ := logger.Set(context.Background(), t.Name())

		accessKey, secretKey := tests.GetAWSKeys(t)
		svc := NewService(nil)
		actual, err := svc.Discover(ctx, accessKey, secretKey)
		require.NoError(t, err)
		expected := []Instance{{
			InstanceID: InstanceID{
				Region:               "eu-west-1",
				DBInstanceIdentifier: "mysql57",
			},
			EndpointAddress:    "mysql57.ckpwzom1xccn.eu-west-1.rds.amazonaws.com",
			EndpointPort:       3306,
			MasterUsername:     "mysql57",
			Engine:             "mysql",
			EngineVersion:      "5.7.19",
			MonitoringInterval: 30 * time.Second,
		}, {
			InstanceID: InstanceID{
				Region:               "us-east-1",
				DBInstanceIdentifier: "aurora1",
			},
			EndpointAddress:    "aurora1.cdy17lilqrl7.us-east-1.rds.amazonaws.com",
			EndpointPort:       3306,
			MasterUsername:     "aurora1",
			Engine:             "aurora",
			EngineVersion:      "5.6.10a",
			MonitoringInterval: 60 * time.Second,
		}, {
			InstanceID: InstanceID{
				Region:               "us-east-1",
				DBInstanceIdentifier: "aurora1-us-east-1c",
			},
			EndpointAddress:    "aurora1-us-east-1c.cdy17lilqrl7.us-east-1.rds.amazonaws.com",
			EndpointPort:       3306,
			MasterUsername:     "aurora1",
			Engine:             "aurora",
			EngineVersion:      "5.6.10a",
			MonitoringInterval: 60 * time.Second,
		}, {
			InstanceID: InstanceID{
				Region:               "us-east-1",
				DBInstanceIdentifier: "mysql56",
			},
			EndpointAddress:    "mysql56.cdy17lilqrl7.us-east-1.rds.amazonaws.com",
			EndpointPort:       3306,
			MasterUsername:     "mysql56",
			Engine:             "mysql",
			EngineVersion:      "5.6.35",
			MonitoringInterval: 15 * time.Second,
		}}
		assert.Equal(t, expected, actual)
	})

	t.Run("WrongKeys", func(t *testing.T) {
		ctx, _ := logger.Set(context.Background(), t.Name())

		accessKey, secretKey := "AKIAIOSFODNN7EXAMPLE", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
		svc := NewService(nil)
		res, err := svc.Discover(ctx, accessKey, secretKey)
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `The security token included in the request is invalid.`), err)
		assert.Empty(t, res)
	})
}

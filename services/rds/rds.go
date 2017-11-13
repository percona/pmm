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
	"net/http"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-managed/utils/logger"
)

// Service is responsible for interactions with AWS RDS.
type Service struct {
	httpClient *http.Client
}

// NewService creates a new service.
func NewService() *Service {
	return &Service{
		httpClient: new(http.Client),
	}
}

type Instance struct {
	ID                 string
	Region             string
	EndpointAddress    string
	EndpointPort       uint16
	MasterUsername     string
	Engine             string
	EngineVersion      string
	MonitoringInterval time.Duration
}

func (svc *Service) Get(ctx context.Context, accessKey, secretKey string) ([]Instance, error) {
	l := logger.Get(ctx).WithField("component", "rds")

	g, ctx := errgroup.WithContext(ctx)
	instances := make(chan Instance)

	for _, r := range endpoints.AwsPartition().Services()[endpoints.RdsServiceID].Regions() {
		regionId := r.ID()
		g.Go(func() error {
			config := aws.Config{
				CredentialsChainVerboseErrors: aws.Bool(true),
				Credentials: credentials.NewCredentials(&credentials.StaticProvider{
					Value: credentials.Value{
						AccessKeyID:     accessKey,
						SecretAccessKey: secretKey,
					},
				}),
				Region:     aws.String(regionId),
				HTTPClient: svc.httpClient,
				Logger:     aws.LoggerFunc(l.Debug),
			}
			if l.Level >= logrus.DebugLevel {
				config.LogLevel = aws.LogLevel(aws.LogDebug)
			}
			s, err := session.NewSessionWithOptions(session.Options{
				Config:            config,
				SharedConfigState: session.SharedConfigDisable,
			})
			if err != nil {
				return errors.WithStack(err)
			}

			out, err := rds.New(s).DescribeDBInstancesWithContext(ctx, new(rds.DescribeDBInstancesInput))
			if err != nil {
				if err2, ok := err.(awserr.Error); ok {
					switch err2.Code() {
					case "InvalidClientTokenId":
						return status.Error(codes.InvalidArgument, err2.Message())
					}
				} else {
					return errors.WithStack(err)
				}
			}
			for _, db := range out.DBInstances {
				instances <- Instance{
					ID:                 *db.DBInstanceIdentifier,
					Region:             regionId,
					EndpointAddress:    *db.Endpoint.Address,
					EndpointPort:       uint16(*db.Endpoint.Port),
					MasterUsername:     *db.MasterUsername,
					Engine:             *db.Engine,
					EngineVersion:      *db.EngineVersion,
					MonitoringInterval: time.Duration(*db.MonitoringInterval) * time.Second,
				}
			}
			return nil
		})
	}

	go func() {
		g.Wait()
		close(instances)
	}()

	var res []Instance
	for i := range instances {
		res = append(res, i)
	}
	sort.Slice(res, func(i, j int) bool {
		if res[i].Region != res[j].Region {
			return res[i].Region < res[j].Region
		}
		return res[i].ID < res[j].ID
	})
	return res, g.Wait()
}

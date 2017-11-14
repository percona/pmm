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
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/logger"
)

// Service is responsible for interactions with AWS RDS.
type Service struct {
	db         *reform.DB
	httpClient *http.Client
}

// NewService creates a new service.
func NewService(db *reform.DB) *Service {
	return &Service{
		db:         db,
		httpClient: new(http.Client),
	}
}

// InstanceID uniquely identifies RDS instance.
// http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Overview.DBInstance.html
// Each DB instance has a DB instance identifier. This customer-supplied name uniquely identifies the DB instance when interacting
// with the Amazon RDS API and AWS CLI commands. The DB instance identifier must be unique for that customer in an AWS Region.
type InstanceID struct {
	Region               string
	DBInstanceIdentifier string
}

type Instance struct {
	InstanceID
	EndpointAddress    string
	EndpointPort       uint16
	MasterUsername     string
	Engine             string
	EngineVersion      string
	MonitoringInterval time.Duration
}

func (svc *Service) Discover(ctx context.Context, accessKey, secretKey string) ([]Instance, error) {
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
					case "InvalidClientTokenId", "EmptyStaticCreds":
						return status.Error(codes.InvalidArgument, err2.Message())
					default:
						return err2
					}
				} else {
					return errors.WithStack(err)
				}
			}
			for _, db := range out.DBInstances {
				instances <- Instance{
					InstanceID: InstanceID{
						Region:               regionId,
						DBInstanceIdentifier: *db.DBInstanceIdentifier,
					},
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
		return res[i].DBInstanceIdentifier < res[j].DBInstanceIdentifier
	})
	return res, g.Wait()
}

func (svc *Service) Add(ctx context.Context, accessKey, secretKey string, ids []InstanceID) error {
	instances, err := svc.Discover(ctx, accessKey, secretKey)
	if err != nil {
		return err
	}

	var add []Instance
	for _, instance := range instances {
		for _, id := range ids {
			if instance.InstanceID == id {
				add = append(add, instance)
			}
		}
	}
	if len(add) == 0 {
		return nil
	}

	tx, err := svc.db.Begin()
	if err != nil {
		return errors.WithStack(err)
	}
	var committed bool
	defer func() {
		if !committed {
			tx.Rollback()
		}
	}()

	for _, instance := range add {
		node := &models.RDSNode{
			Type: models.RDSNodeType,

			Region:   &instance.Region,
			Hostname: &instance.DBInstanceIdentifier,
		}
		if err = tx.Save(node); err != nil {
			return err
		}

		service := &models.RDSService{
			Type:   models.RDSServiceType,
			NodeID: node.ID,
		}
		if err = tx.Save(service); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

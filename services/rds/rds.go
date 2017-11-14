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

	"github.com/AlekSi/pointer"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/mattn/go-sqlite3"
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
	Region string
	Name   string // DBInstanceIdentifier
}

type Instance struct {
	Node    models.RDSNode
	Service models.RDSService
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
				if err, ok := err.(awserr.Error); ok {
					switch err.Code() {
					case "InvalidClientTokenId", "EmptyStaticCreds":
						return status.Error(codes.InvalidArgument, err.Message())
					default:
						return err
					}
				} else {
					return errors.WithStack(err)
				}
			}
			for _, db := range out.DBInstances {
				instances <- Instance{
					Node: models.RDSNode{
						Type: models.RDSNodeType,

						Name:   *db.DBInstanceIdentifier,
						Region: regionId,
					},
					Service: models.RDSService{
						Type: models.RDSServiceType,

						Address:       db.Endpoint.Address,
						Port:          pointer.ToUint16(uint16(*db.Endpoint.Port)),
						Engine:        db.Engine,
						EngineVersion: db.EngineVersion,
					},
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
		if res[i].Node.Region != res[j].Node.Region {
			return res[i].Node.Region < res[j].Node.Region
		}
		return res[i].Node.Name < res[j].Node.Name
	})
	return res, g.Wait()
}

func (svc *Service) List(ctx context.Context) ([]Instance, error) {
	var res []Instance
	err := svc.db.InTransaction(func(tx *reform.TX) error {
		structs, e := tx.SelectAllFrom(models.RDSNodeTable, "WHERE type = ? ORDER BY id", models.RDSNodeType)
		if e != nil {
			return e
		}
		nodes := make([]models.RDSNode, len(structs))
		for i, str := range structs {
			nodes[i] = *str.(*models.RDSNode)
		}

		structs, e = tx.SelectAllFrom(models.RDSServiceTable, "WHERE type = ? ORDER BY id", models.RDSServiceType)
		if e != nil {
			return e
		}
		services := make([]models.RDSService, len(structs))
		for i, str := range structs {
			services[i] = *str.(*models.RDSService)
		}

		for _, node := range nodes {
			for _, service := range services {
				if node.ID == service.NodeID {
					res = append(res, Instance{
						Node:    node,
						Service: service,
					})
				}
			}
		}
		return nil
	})
	return res, err
}

func (svc *Service) Add(ctx context.Context, accessKey, secretKey string, ids []InstanceID) error {
	instances, err := svc.Discover(ctx, accessKey, secretKey)
	if err != nil {
		return err
	}

	var add []Instance
	for _, instance := range instances {
		for _, id := range ids {
			if instance.Node.Name == id.Name && instance.Node.Region == id.Region {
				add = append(add, instance)
			}
		}
	}
	if len(add) == 0 {
		return nil
	}

	return svc.db.InTransaction(func(tx *reform.TX) error {
		for _, instance := range add {
			node := &models.RDSNode{
				Type: models.RDSNodeType,
				Name: instance.Node.Name,

				Region: instance.Node.Region,
			}
			if e := tx.Insert(node); e != nil {
				if e, ok := e.(sqlite3.Error); ok && e.ExtendedCode == sqlite3.ErrConstraintUnique {
					return status.Errorf(codes.AlreadyExists, "RDS instance %q already exists in region %q.",
						node.Name, node.Region)
				}
				return errors.WithStack(e)
			}

			service := &models.RDSService{
				Type:   models.RDSServiceType,
				NodeID: node.ID,

				Address:       instance.Service.Address,
				Port:          instance.Service.Port,
				Engine:        instance.Service.Engine,
				EngineVersion: instance.Service.EngineVersion,
			}
			if e := tx.Insert(service); e != nil {
				return errors.WithStack(e)
			}

			// TODO insert agents

			// TODO start agents
		}

		return nil
	})
}

func (svc *Service) Remove(ctx context.Context, ids []InstanceID) error {
	if len(ids) == 0 {
		return nil
	}

	return svc.db.InTransaction(func(tx *reform.TX) error {
		for _, instance := range ids {
			var node models.RDSNode
			if e := tx.SelectOneTo(&node, "WHERE type = ? AND name = ? AND region = ?", models.RDSNodeType, instance.Name, instance.Region); e != nil {
				if e == reform.ErrNoRows {
					return status.Errorf(codes.NotFound, "RDS instance %q not found in region %q.",
						instance.Name, instance.Region)
				}
				return errors.WithStack(e)
			}

			// TODO stop agents

			// TODO delete agents

			if _, e := tx.DeleteFrom(models.RDSServiceTable, "WHERE node_id = ?", node.ID); e != nil {
				return errors.WithStack(e)
			}

			if e := tx.Delete(&node); e != nil {
				return errors.WithStack(e)
			}
		}

		return nil
	})
}

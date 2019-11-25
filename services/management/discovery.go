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

package management

import (
	"context"
	"net"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/percona/pmm/api/managementpb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/logger"
)

const (
	// maximum time for AWS discover APIs calls
	awsDiscoverTimeout = 7 * time.Second
)

// DiscoveryService represents instance discovery service.
type DiscoveryService struct {
	db *reform.DB
}

// NewDiscoveryService creates new instance discovery service.
func NewDiscoveryService(db *reform.DB) *DiscoveryService {
	return &DiscoveryService{
		db: db,
	}
}

var (
	rdsEngines = map[string]managementpb.DiscoverRDSEngine{
		"mysql":  managementpb.DiscoverRDSEngine_DISCOVER_RDS_MYSQL,
		"aurora": managementpb.DiscoverRDSEngine_DISCOVER_RDS_MYSQL, // TODO what value AWS returns for Aurora for PostgreSQL?
	}
	rdsEnginesKeys = []*string{
		pointer.ToString("mysql"),
		pointer.ToString("aurora"),
	}
)

// discoverRDSRegion returns a list of RDS instances from a single region.
// Returned error is wrapped with a stack trace, but unchanged otherwise.
//nolint:interfacer
func discoverRDSRegion(ctx context.Context, sess *session.Session, region string) ([]*rds.DBInstance, error) {
	var res []*rds.DBInstance
	input := &rds.DescribeDBInstancesInput{
		Filters: []*rds.Filter{{
			Name:   pointer.ToString("engine"),
			Values: rdsEnginesKeys,
		}},
	}
	fn := func(out *rds.DescribeDBInstancesOutput, lastPage bool) bool {
		res = append(res, out.DBInstances...)
		return true // continue pagination
	}
	err := rds.New(sess, &aws.Config{Region: &region}).DescribeDBInstancesPagesWithContext(ctx, input, fn)
	if err != nil {
		return res, errors.WithStack(err)
	}
	return res, nil
}

// listRegions returns a list of AWS regions for given partitions.
func listRegions(partitions []string) []string {
	set := make(map[string]struct{})
	for _, p := range partitions {
		for _, partition := range endpoints.DefaultPartitions() {
			if p != partition.ID() {
				continue
			}

			for r := range partition.Services()[endpoints.RdsServiceID].Regions() {
				set[r] = struct{}{}
			}
			break
		}
	}

	slice := make([]string, 0, len(set))
	for r := range set {
		slice = append(slice, r)
	}
	sort.Strings(slice)

	return slice
}

// DiscoverRDS returns a list of RDS instances from all regions in configured AWS partitions.
func (s *DiscoveryService) DiscoverRDS(ctx context.Context, req *managementpb.DiscoverRDSRequest) (*managementpb.DiscoverRDSResponse, error) {
	l := logger.Get(ctx).WithField("component", "discover/rds")

	settings, err := models.GetSettings(s.db.Querier)
	if err != nil {
		return nil, err
	}

	// use given credentials, or default credential chain
	var creds *credentials.Credentials
	if req.AwsAccessKey != "" || req.AwsSecretKey != "" {
		creds = credentials.NewStaticCredentials(req.AwsAccessKey, req.AwsSecretKey, "")
	}
	cfg := &aws.Config{
		CredentialsChainVerboseErrors: aws.Bool(true),
		Credentials:                   creds,
		HTTPClient:                    new(http.Client),
	}
	if l.Logger.GetLevel() >= logrus.DebugLevel {
		cfg.LogLevel = aws.LogLevel(aws.LogDebug)
	}
	sess, err := session.NewSession(cfg)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// do not break our API if some AWS region is slow or down
	ctx, cancel := context.WithTimeout(ctx, awsDiscoverTimeout)
	defer cancel()
	var wg errgroup.Group
	instances := make(chan *managementpb.DiscoverRDSInstance)

	for _, region := range listRegions(settings.AWSPartitions) {
		region := region
		wg.Go(func() error {
			regInstances, err := discoverRDSRegion(ctx, sess, region)
			if err != nil {
				l.Debugf("%s: %+v", region, err)
			}

			for _, db := range regInstances {
				l.Debugf("Discovered instance: %+v", db)

				instances <- &managementpb.DiscoverRDSInstance{
					Region:     region,
					InstanceId: *db.DBInstanceIdentifier,
					Address: net.JoinHostPort(
						pointer.GetString(db.Endpoint.Address),
						strconv.FormatInt(pointer.GetInt64(db.Endpoint.Port), 10),
					),
					Engine:        rdsEngines[*db.Engine],
					EngineVersion: pointer.GetString(db.EngineVersion),
				}
			}

			return err
		})
	}

	go func() {
		_ = wg.Wait() // checked below
		close(instances)
	}()

	res := new(managementpb.DiscoverRDSResponse)
	for i := range instances {
		res.RdsInstances = append(res.RdsInstances, i)
	}

	// sort by region and id
	sort.Slice(res.RdsInstances, func(i, j int) bool {
		if res.RdsInstances[i].Region != res.RdsInstances[j].Region {
			return res.RdsInstances[i].Region < res.RdsInstances[j].Region
		}
		return res.RdsInstances[i].InstanceId < res.RdsInstances[j].InstanceId
	})

	// ignore error if there are some results
	if len(res.RdsInstances) > 0 {
		return res, nil
	}

	// return better gRPC errors in typical cases
	err = wg.Wait()
	if e, ok := errors.Cause(err).(awserr.Error); ok {
		switch {
		case e.Code() == "InvalidClientTokenId":
			return res, status.Error(codes.InvalidArgument, e.Message())
		case e.OrigErr() == context.Canceled || e.OrigErr() == context.DeadlineExceeded:
			return res, status.Error(codes.DeadlineExceeded, "Request timeout.")
		default:
			return res, status.Error(codes.Unknown, e.Error())
		}
	}
	return nil, err
}

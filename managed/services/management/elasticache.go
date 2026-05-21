// Copyright (C) 2023 Percona LLC
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
	"net/http"
	"sort"

	"github.com/AlekSi/pointer"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	ectypes "github.com/aws/aws-sdk-go-v2/service/elasticache/types"
	"github.com/aws/smithy-go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/utils/logger"
)

// elasticacheEngines maps AWS engine names to our proto enum.
var elasticacheEngines = map[string]managementv1.DiscoverElastiCacheEngine{
	"redis":  managementv1.DiscoverElastiCacheEngine_DISCOVER_ELASTI_CACHE_ENGINE_REDIS,
	"valkey": managementv1.DiscoverElastiCacheEngine_DISCOVER_ELASTI_CACHE_ENGINE_VALKEY,
}

// discoverElastiCacheRegion returns a list of ElastiCache replication groups from a single region.
func discoverElastiCacheRegion(ctx context.Context, cfg aws.Config, region string) ([]ectypes.ReplicationGroup, error) {
	var res []ectypes.ReplicationGroup
	client := elasticache.NewFromConfig(cfg, func(o *elasticache.Options) {
		o.Region = region
	})

	paginator := elasticache.NewDescribeReplicationGroupsPaginator(client, &elasticache.DescribeReplicationGroupsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return res, err
		}
		res = append(res, page.ReplicationGroups...)
	}

	return res, nil
}

// listElastiCacheRegions returns a list of AWS regions where ElastiCache is available.
// ElastiCache is available in the same regions as RDS, so we reuse the same list.
func listElastiCacheRegions(partitions []string) []string {
	return listRegions(partitions)
}

// DiscoverElastiCache discovers ElastiCache replication groups (Valkey/Redis).
func (s *ManagementService) DiscoverElastiCache(ctx context.Context, req *managementv1.DiscoverElastiCacheRequest) (*managementv1.DiscoverElastiCacheResponse, error) { //nolint:gocognit
	l := logger.Get(ctx).WithField("component", "discover/elasticache")

	settings, err := models.GetSettings(s.db.Querier)
	if err != nil {
		return nil, err
	}

	// Use given credentials, or default credential chain.
	var creds aws.CredentialsProvider
	if req.AwsAccessKey != "" && req.AwsSecretKey != "" {
		creds = credentials.NewStaticCredentialsProvider(req.AwsAccessKey, req.AwsSecretKey, "")
	}

	opts := []func(*config.LoadOptions) error{
		config.WithCredentialsProvider(creds),
		config.WithHTTPClient(&http.Client{}),
	}
	if l.Logger != nil && l.Logger.Level >= logrus.DebugLevel {
		opts = append(opts, config.WithClientLogMode(aws.LogRetries|aws.LogRequestWithBody|aws.LogResponseWithBody))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Default to standard AWS partition if none configured.
	partitions := settings.AWSPartitions
	if len(partitions) == 0 {
		partitions = []string{"aws"}
	}

	ctx, cancel := context.WithTimeout(ctx, awsDiscoverTimeout)
	defer cancel()
	var wg errgroup.Group
	instances := make(chan *managementv1.DiscoverElastiCacheInstance)

	for _, region := range listElastiCacheRegions(partitions) {
		wg.Go(func() error {
			regGroups, err := discoverElastiCacheRegion(ctx, cfg, region)
			if err != nil {
				l.Debugf("%s: %+v", region, err)
			}

			for _, rg := range regGroups {
				if rg.Status == nil || *rg.Status != "available" {
					continue
				}

				engine, ok := elasticacheEngines[pointer.GetString(rg.Engine)]
				if !ok {
					continue
				}

				clusterName := pointer.GetString(rg.ReplicationGroupId)
				nodeType := pointer.GetString(rg.CacheNodeType)
				transitEncryption := rg.TransitEncryptionEnabled != nil && *rg.TransitEncryptionEnabled

				// Cluster Mode Enabled: use ConfigurationEndpoint.
				if rg.ClusterEnabled != nil && *rg.ClusterEnabled && rg.ConfigurationEndpoint != nil {
					az := ""
					if len(rg.NodeGroups) > 0 && len(rg.NodeGroups[0].NodeGroupMembers) > 0 {
						az = pointer.GetString(rg.NodeGroups[0].NodeGroupMembers[0].PreferredAvailabilityZone)
					}
					instances <- &managementv1.DiscoverElastiCacheInstance{
						Region:                   region,
						Az:                       az,
						InstanceId:               clusterName,
						NodeModel:                nodeType,
						Address:                  pointer.GetString(rg.ConfigurationEndpoint.Address),
						Port:                     uint32(pointer.GetInt32(rg.ConfigurationEndpoint.Port)), //nolint:gosec
						Engine:                   engine,
						TransitEncryptionEnabled: transitEncryption,
						Cluster:                  clusterName,
					}
					continue
				}

				// Cluster Mode Disabled: report per-shard endpoints.
				for _, ng := range rg.NodeGroups {
					if ng.PrimaryEndpoint == nil {
						continue
					}

					az := ""
					if len(ng.NodeGroupMembers) > 0 {
						az = pointer.GetString(ng.NodeGroupMembers[0].PreferredAvailabilityZone)
					}

					instances <- &managementv1.DiscoverElastiCacheInstance{
						Region:                   region,
						Az:                       az,
						InstanceId:               clusterName,
						NodeModel:                nodeType,
						Address:                  pointer.GetString(ng.PrimaryEndpoint.Address),
						Port:                     uint32(pointer.GetInt32(ng.PrimaryEndpoint.Port)), //nolint:gosec
						Engine:                   engine,
						TransitEncryptionEnabled: transitEncryption,
						Cluster:                  clusterName,
					}
				}
			}

			return err
		})
	}

	go func() {
		_ = wg.Wait()
		close(instances)
	}()

	res := &managementv1.DiscoverElastiCacheResponse{}
	for i := range instances {
		res.ElasticacheInstances = append(res.ElasticacheInstances, i)
	}

	sort.Slice(res.ElasticacheInstances, func(i, j int) bool {
		if res.ElasticacheInstances[i].Region != res.ElasticacheInstances[j].Region {
			return res.ElasticacheInstances[i].Region < res.ElasticacheInstances[j].Region
		}
		return res.ElasticacheInstances[i].InstanceId < res.ElasticacheInstances[j].InstanceId
	})

	if len(res.ElasticacheInstances) != 0 {
		return res, nil
	}

	// Return better gRPC errors in typical cases.
	err = wg.Wait()
	if err != nil {
		var apiErr *smithy.GenericAPIError
		if errors.As(err, &apiErr) {
			switch {
			case apiErr.Code == "InvalidClientTokenId":
				return res, status.Error(codes.InvalidArgument, apiErr.Message)
			case errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded):
				return res, status.Error(codes.DeadlineExceeded, "Request timeout.")
			default:
				return res, status.Error(codes.Unknown, apiErr.Error())
			}
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return res, status.Error(codes.DeadlineExceeded, "Request timeout.")
		}

		return res, status.Error(codes.Unknown, err.Error())
	}

	return res, nil
}

// addElastiCache adds an ElastiCache instance as a Valkey service.
func (s *ManagementService) addElastiCache(ctx context.Context, req *managementv1.AddElastiCacheServiceParams) (*managementv1.AddServiceResponse, error) {
	ec := &managementv1.ElastiCacheServiceResult{}

	pmmAgentID := models.PMMServerAgentID
	if req.GetPmmAgentId() != "" {
		pmmAgentID = req.GetPmmAgentId()
	}

	errTx := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		if req.NodeName == "" {
			req.NodeName = req.InstanceId
		}
		if req.ServiceName == "" {
			req.ServiceName = req.InstanceId
		}

		node, err := models.CreateNode(tx.Querier, models.RemoteElastiCacheNodeType, &models.CreateNodeParams{
			NodeName:     req.NodeName,
			NodeModel:    req.NodeModel,
			AZ:           req.Az,
			InstanceID:   req.InstanceId,
			Address:      req.Address,
			Region:       &req.Region,
			CustomLabels: req.CustomLabels,
		})
		if err != nil {
			return err
		}
		invNode, err := services.ToAPINode(node)
		if err != nil {
			return err
		}
		ec.Node = invNode.(*inventoryv1.RemoteElastiCacheNode) //nolint:forcetypeassert

		metricsMode, err := supportedMetricsMode(req.MetricsMode, pmmAgentID)
		if err != nil {
			return err
		}

		service, err := models.AddNewService(tx.Querier, models.ValkeyServiceType, &models.AddDBMSServiceParams{
			ServiceName:    req.ServiceName,
			NodeID:         node.NodeID,
			Environment:    req.Environment,
			Cluster:        req.Cluster,
			ReplicationSet: req.ReplicationSet,
			CustomLabels:   req.CustomLabels,
			Address:        &req.Address,
			Port:           pointer.ToUint16(uint16(req.Port)), //nolint:gosec,modernize
		})
		if err != nil {
			return err
		}
		invService, err := services.ToAPIService(service)
		if err != nil {
			return err
		}
		ec.ValkeyService = invService.(*inventoryv1.ValkeyService) //nolint:forcetypeassert

		valkeyExporter, err := models.CreateAgent(tx.Querier, models.ValkeyExporterType, &models.CreateAgentParams{
			PMMAgentID:    pmmAgentID,
			ServiceID:     service.ServiceID,
			Username:      req.Username,
			Password:      req.Password,
			TLS:           req.Tls,
			TLSSkipVerify: req.TlsSkipVerify,
			ExporterOptions: models.ExporterOptions{
				PushMetrics: isPushMode(metricsMode),
			},
		})
		if err != nil {
			return err
		}

		if !req.SkipConnectionCheck {
			if err = s.cc.CheckConnectionToService(ctx, tx.Querier, service, valkeyExporter); err != nil {
				return err
			}
			if err = s.sib.GetInfoFromService(ctx, tx.Querier, service, valkeyExporter); err != nil {
				return err
			}
		}

		invAgent, err := services.ToAPIAgent(tx.Querier, valkeyExporter)
		if err != nil {
			return err
		}
		ec.ValkeyExporter = invAgent.(*inventoryv1.ValkeyExporter) //nolint:forcetypeassert

		return nil
	})

	if errTx != nil {
		return nil, errTx
	}

	s.state.RequestStateUpdate(ctx, pmmAgentID)

	return &managementv1.AddServiceResponse{
		Service: &managementv1.AddServiceResponse_Elasticache{
			Elasticache: ec,
		},
	}, nil
}

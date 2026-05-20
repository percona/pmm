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
	"fmt"
	"net/http"
	"sort"
	"sync/atomic"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	ectypes "github.com/aws/aws-sdk-go-v2/service/elasticache/types"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

const (
	// How long to wait after startup before first discovery run.
	elasticacheDiscoveryStartupDelay = 30 * time.Second
	// How often to run the reconciliation loop.
	elasticacheDiscoveryInterval = 5 * time.Minute
	// Timeout for a single discovery cycle (all regions).
	elasticacheDiscoveryTimeout = 60 * time.Second
	// Tag that must be "true" on a replication group for it to be auto-added.
	elasticacheTagKey   = "pmm_enable"
	elasticacheTagValue = "true"
	// Label used to identify services managed by auto-discovery.
	elasticacheManagedByLabel = "elasticache-autodiscovery"
)

// ElastiCacheDiscovery is a background service that periodically discovers
// ElastiCache replication groups tagged with pmm_enable=true and reconciles
// them with the PMM inventory.
type ElastiCacheDiscovery struct {
	db    *reform.DB
	state agentsStateUpdater
	l     *logrus.Entry
}

// NewElastiCacheDiscovery creates a new ElastiCacheDiscovery service.
func NewElastiCacheDiscovery(db *reform.DB, state agentsStateUpdater) *ElastiCacheDiscovery {
	return &ElastiCacheDiscovery{
		db:    db,
		state: state,
		l:     logrus.WithField("component", "elasticache-discovery"),
	}
}

// discoveredInstance holds info about an ElastiCache endpoint discovered from AWS.
type discoveredInstance struct {
	Region      string
	AZ          string
	ClusterID   string // ReplicationGroupId
	NodeType    string // CacheNodeType
	Address     string
	Port        int32
	Engine      string // "redis" or "valkey"
	TLS         bool
	Environment string // from AWS "Environment" tag
	Role        string // "primary", "reader", or "cluster" for cluster mode
}

// Run starts the background discovery loop. It blocks until ctx is cancelled.
func (d *ElastiCacheDiscovery) Run(ctx context.Context) {
	d.l.Info("Starting (waiting for initial delay)...")
	select {
	case <-time.After(elasticacheDiscoveryStartupDelay):
	case <-ctx.Done():
		return
	}

	d.l.Info("Started.")
	ticker := time.NewTicker(elasticacheDiscoveryInterval)
	defer ticker.Stop()

	for {
		d.reconcile(ctx)

		select {
		case <-ticker.C:
		case <-ctx.Done():
			d.l.Info("Stopped.")
			return
		}
	}
}

// reconcile runs a single discovery + sync cycle.
func (d *ElastiCacheDiscovery) reconcile(ctx context.Context) {
	d.l.Info("Running reconciliation...")

	settings, err := models.GetSettings(d.db.Querier)
	if err != nil {
		d.l.Warnf("Failed to get settings: %v", err)
		return
	}

	// Default to standard AWS partition if none configured.
	partitions := settings.AWSPartitions
	if len(partitions) == 0 {
		partitions = []string{"aws"}
	}

	regions := listElastiCacheRegions(partitions)
	discovered, scanComplete, err := d.discoverTaggedInstances(ctx, regions)
	if err != nil {
		d.l.Warnf("Discovery failed: %v", err)
		return
	}

	managed, err := d.findManagedServices()
	if err != nil {
		d.l.Warnf("Failed to list managed services: %v", err)
		return
	}

	d.l.Infof("Discovered %d endpoint(s), PMM has %d managed service(s)", len(discovered), len(managed))

	// Build maps keyed by address for diffing.
	expectedByAddr := make(map[string]discoveredInstance, len(discovered))
	for _, inst := range discovered {
		expectedByAddr[inst.Address] = inst
	}

	managedByAddr := make(map[string]*models.Service, len(managed))
	for _, svc := range managed {
		if svc.Address != nil {
			managedByAddr[*svc.Address] = svc
		}
	}

	// Add missing. Adds are safe even on a partial scan: any tagged cluster we
	// did see is still legitimately tagged, and the operation is idempotent.
	var added, addFailed int
	for addr, inst := range expectedByAddr {
		if _, exists := managedByAddr[addr]; exists {
			continue
		}
		if err := d.addInstance(ctx, inst); err != nil {
			d.l.Warnf("Failed to add %s (%s): %v", inst.ClusterID, addr, err)
			addFailed++
			continue
		}
		added++
	}

	// Remove stale — only when the AWS scan completed cleanly across every
	// region and tag lookup. A partial scan would make legitimate services look
	// "missing" (e.g. throttled ListTagsForResource → cluster appears untagged)
	// and trigger false removals.
	var removed int
	if !scanComplete {
		d.l.Warnf("Discovery scan was incomplete; skipping stale-removal this cycle")
	} else {
		for addr, svc := range managedByAddr {
			if _, exists := expectedByAddr[addr]; exists {
				continue
			}
			if err := d.removeService(ctx, svc); err != nil {
				d.l.Warnf("Failed to remove %s (%s): %v", svc.ServiceName, addr, err)
				continue
			}
			removed++
		}
	}

	unchanged := len(expectedByAddr) - added - addFailed
	d.l.Infof("Reconciliation complete: +%d added, -%d removed, =%d unchanged, %d failed (scan complete: %t)",
		added, removed, unchanged, addFailed, scanComplete)
}

// discoverTaggedInstances discovers ElastiCache replication groups across regions
// and filters to those tagged with pmm_enable=true.
//
// Returns (instances, scanComplete, error). scanComplete is false when any
// region scan or per-cluster tag lookup failed; callers must not use the result
// to drive destructive operations (e.g. stale-removal) in that case.
func (d *ElastiCacheDiscovery) discoverTaggedInstances(ctx context.Context, regions []string) ([]discoveredInstance, bool, error) {
	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithHTTPClient(&http.Client{}),
	)
	if err != nil {
		return nil, false, fmt.Errorf("failed to load AWS config: %w", err)
	}

	d.l.Debugf("Scanning %d region(s)", len(regions))

	ctx, cancel := context.WithTimeout(ctx, elasticacheDiscoveryTimeout)
	defer cancel()

	var wg errgroup.Group
	results := make(chan discoveredInstance)
	var scanFailed atomic.Bool

	for _, region := range regions {
		wg.Go(func() error {
			instances, complete := d.discoverRegionTagged(ctx, cfg, region)
			if !complete {
				scanFailed.Store(true)
			}
			if len(instances) > 0 {
				d.l.Debugf("Region %s: found %d tagged endpoint(s)", region, len(instances))
			}
			for _, inst := range instances {
				results <- inst
			}
			return nil
		})
	}

	go func() {
		_ = wg.Wait()
		close(results)
	}()

	var discovered []discoveredInstance
	for inst := range results {
		discovered = append(discovered, inst)
	}

	sort.Slice(discovered, func(i, j int) bool {
		if discovered[i].Region != discovered[j].Region {
			return discovered[i].Region < discovered[j].Region
		}
		return discovered[i].Address < discovered[j].Address
	})

	return discovered, !scanFailed.Load(), nil
}

// discoverRegionTagged returns ElastiCache replication groups in a region that are tagged pmm_enable=true.
//
// The second return value is true only when every AWS call in this region
// succeeded. A false value means the caller should not trust the absence of any
// previously-known cluster from the returned slice (e.g. throttled
// DescribeReplicationGroups or ListTagsForResource).
func (d *ElastiCacheDiscovery) discoverRegionTagged(ctx context.Context, cfg aws.Config, region string) ([]discoveredInstance, bool) {
	client := elasticache.NewFromConfig(cfg, func(o *elasticache.Options) {
		o.Region = region
	})

	var instances []discoveredInstance
	complete := true

	paginator := elasticache.NewDescribeReplicationGroupsPaginator(client, &elasticache.DescribeReplicationGroupsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			d.l.Warnf("Region %s: DescribeReplicationGroups failed: %v", region, err)
			return instances, false
		}

		for _, rg := range page.ReplicationGroups {
			if rg.Status == nil || *rg.Status != "available" {
				continue
			}

			engine := pointer.GetString(rg.Engine)
			if engine != "redis" && engine != "valkey" {
				continue
			}

			clusterName := pointer.GetString(rg.ReplicationGroupId)

			// Skip clusters with authentication enabled (no credentials support).
			if rg.AuthTokenEnabled != nil && *rg.AuthTokenEnabled {
				d.l.Debugf("Skipping %s: AUTH token enabled", clusterName)
				continue
			}
			if len(rg.UserGroupIds) > 0 {
				d.l.Debugf("Skipping %s: ACL user groups configured", clusterName)
				continue
			}

			// Check tags for pmm_enable=true and extract Environment.
			// On error we mark the region as incomplete and skip this cluster:
			// we can't tell whether it's tagged, and silently treating the
			// unknown as "not tagged" would let stale-removal delete a
			// legitimately-managed service.
			tags, err := d.checkTags(ctx, client, rg)
			if err != nil {
				d.l.Warnf("Region %s cluster %s: ListTagsForResource failed: %v", region, clusterName, err)
				complete = false
				continue
			}
			if !tags.enabled {
				continue
			}

			nodeType := pointer.GetString(rg.CacheNodeType)
			tls := rg.TransitEncryptionEnabled != nil && *rg.TransitEncryptionEnabled

			// Cluster Mode Enabled: use the ConfigurationEndpoint (covers all shards).
			if rg.ClusterEnabled != nil && *rg.ClusterEnabled && rg.ConfigurationEndpoint != nil {
				az := ""
				if len(rg.NodeGroups) > 0 && len(rg.NodeGroups[0].NodeGroupMembers) > 0 {
					az = pointer.GetString(rg.NodeGroups[0].NodeGroupMembers[0].PreferredAvailabilityZone)
				}

				instances = append(instances, discoveredInstance{
					Region:      region,
					AZ:          az,
					ClusterID:   clusterName,
					NodeType:    nodeType,
					Address:     pointer.GetString(rg.ConfigurationEndpoint.Address),
					Port:        pointer.GetInt32(rg.ConfigurationEndpoint.Port),
					Engine:      engine,
					TLS:         tls,
					Environment: tags.environment,
					Role:        "cluster",
				})
				continue
			}

			// Cluster Mode Disabled: add primary (writer) and reader endpoints per shard.
			for _, ng := range rg.NodeGroups {
				az := ""
				if len(ng.NodeGroupMembers) > 0 {
					az = pointer.GetString(ng.NodeGroupMembers[0].PreferredAvailabilityZone)
				}

				if ng.PrimaryEndpoint != nil {
					instances = append(instances, discoveredInstance{
						Region:      region,
						AZ:          az,
						ClusterID:   clusterName,
						NodeType:    nodeType,
						Address:     pointer.GetString(ng.PrimaryEndpoint.Address),
						Port:        pointer.GetInt32(ng.PrimaryEndpoint.Port),
						Engine:      engine,
						TLS:         tls,
						Environment: tags.environment,
						Role:        "primary",
					})
				}

				if ng.ReaderEndpoint != nil {
					instances = append(instances, discoveredInstance{
						Region:      region,
						AZ:          az,
						ClusterID:   clusterName,
						NodeType:    nodeType,
						Address:     pointer.GetString(ng.ReaderEndpoint.Address),
						Port:        pointer.GetInt32(ng.ReaderEndpoint.Port),
						Engine:      engine,
						TLS:         tls,
						Environment: tags.environment,
						Role:        "reader",
					})
				}
			}
		}
	}

	return instances, complete
}

// tagResult holds the result of a tag check.
type tagResult struct {
	enabled     bool
	environment string
}

// checkTags checks if a replication group has the pmm_enable=true tag and extracts the Environment tag.
//
// A non-nil error means the AWS call failed; callers must treat the result as
// unknown (not as "not tagged") to avoid false stale-removals.
func (d *ElastiCacheDiscovery) checkTags(ctx context.Context, client *elasticache.Client, rg ectypes.ReplicationGroup) (tagResult, error) {
	if rg.ARN == nil {
		return tagResult{}, nil
	}

	resp, err := client.ListTagsForResource(ctx, &elasticache.ListTagsForResourceInput{
		ResourceName: rg.ARN,
	})
	if err != nil {
		return tagResult{}, fmt.Errorf("list tags for %s: %w", pointer.GetString(rg.ReplicationGroupId), err)
	}

	result := tagResult{}
	for _, tag := range resp.TagList {
		key := pointer.GetString(tag.Key)
		value := pointer.GetString(tag.Value)
		if key == elasticacheTagKey && value == elasticacheTagValue {
			result.enabled = true
		}
		if key == "Environment" {
			result.environment = value
		}
	}
	return result, nil
}

// findManagedServices returns all Valkey services in the inventory that were added by auto-discovery.
func (d *ElastiCacheDiscovery) findManagedServices() ([]*models.Service, error) {
	valkeyType := models.ValkeyServiceType
	allServices, err := models.FindServices(d.db.Querier, models.ServiceFilters{
		ServiceType: &valkeyType,
	})
	if err != nil {
		return nil, err
	}

	var managed []*models.Service
	for _, svc := range allServices {
		labels, err := svc.GetCustomLabels()
		if err != nil {
			continue
		}
		if labels["managed_by"] == elasticacheManagedByLabel {
			managed = append(managed, svc)
		}
	}
	return managed, nil
}

// addInstance creates a RemoteElastiCacheNode + ValkeyService + ValkeyExporter for a discovered instance.
func (d *ElastiCacheDiscovery) addInstance(ctx context.Context, inst discoveredInstance) error {
	serviceName := fmt.Sprintf("elasticache-%s", inst.ClusterID)
	if inst.Role == "reader" {
		serviceName = fmt.Sprintf("elasticache-%s-reader", inst.ClusterID)
	}
	d.l.Infof("ADD %s [%s] %s:%d (env=%s)", serviceName, inst.Role, inst.Address, inst.Port, inst.Environment)

	pmmAgentID := models.PMMServerAgentID

	return d.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		node, err := models.CreateNode(tx.Querier, models.RemoteElastiCacheNodeType, &models.CreateNodeParams{
			NodeName:   serviceName,
			NodeModel:  inst.NodeType,
			AZ:         inst.AZ,
			InstanceID: inst.ClusterID,
			Address:    inst.Address,
			Region:     &inst.Region,
			CustomLabels: map[string]string{
				"managed_by": elasticacheManagedByLabel,
				"source":     "elasticache",
			},
		})
		if err != nil {
			return fmt.Errorf("create node: %w", err)
		}

		service, err := models.AddNewService(tx.Querier, models.ValkeyServiceType, &models.AddDBMSServiceParams{
			ServiceName: serviceName,
			NodeID:      node.NodeID,
			Environment: inst.Environment,
			Cluster:     inst.ClusterID,
			Address:     &inst.Address,
			Port:        pointer.ToUint16(uint16(inst.Port)), //nolint:gosec
			CustomLabels: map[string]string{
				"managed_by": elasticacheManagedByLabel,
				"source":     "elasticache",
				"engine":     inst.Engine,
				"role":       inst.Role,
			},
		})
		if err != nil {
			return fmt.Errorf("add service: %w", err)
		}

		_, err = models.CreateAgent(tx.Querier, models.ValkeyExporterType, &models.CreateAgentParams{
			PMMAgentID: pmmAgentID,
			ServiceID:  service.ServiceID,
			TLS:        inst.TLS,
			ExporterOptions: models.ExporterOptions{
				PushMetrics: true,
			},
		})
		if err != nil {
			return fmt.Errorf("create agent: %w", err)
		}

		d.state.RequestStateUpdate(ctx, pmmAgentID)
		return nil
	})
}

// removeService removes a service, its agents, and its node.
func (d *ElastiCacheDiscovery) removeService(ctx context.Context, svc *models.Service) error {
	d.l.Infof("DEL %s (%s)", svc.ServiceName, pointer.GetString(svc.Address))

	return d.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		agents, err := models.FindAgents(tx.Querier, models.AgentFilters{ServiceID: svc.ServiceID})
		if err != nil {
			return err
		}
		for _, agent := range agents {
			if _, err := models.RemoveAgent(tx.Querier, agent.AgentID, models.RemoveRestrict); err != nil {
				return err
			}
			if agent.PMMAgentID != nil {
				d.state.RequestStateUpdate(ctx, pointer.GetString(agent.PMMAgentID))
			}
		}

		nodeID := svc.NodeID

		if err := models.RemoveService(tx.Querier, svc.ServiceID, models.RemoveCascade); err != nil {
			return err
		}

		node, err := models.FindNodeByID(tx.Querier, nodeID)
		if err != nil {
			return err
		}
		if node.NodeType == models.RemoteElastiCacheNodeType {
			if err := models.RemoveNode(tx.Querier, node.NodeID, models.RemoveCascade); err != nil {
				return err
			}
		}

		return nil
	})
}

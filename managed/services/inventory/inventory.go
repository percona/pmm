package inventory

import (
	"context"
	"fmt"
	"github.com/AlekSi/pointer"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/agents"
	prom "github.com/prometheus/client_golang/prometheus"
	"gopkg.in/reform.v1"
	"strconv"
	"time"
)

const (
	cancelTime                  = 3 * time.Second
	serviceEnabled      float64 = 1
	prometheusNamespace         = "pmm_managed"
	prometheusSubsystem         = "inventory"
)

var (
	mAgentsDesc = prom.NewDesc(
		prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "agents"),
		"The current information about agent",
		[]string{"agent_type", "service_id", "node_id", "pmm_agent_id", "disabled", "version"},
		nil)
	mNodesDesc = prom.NewDesc(
		prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "nodes"),
		"The current information about node",
		[]string{"node_type", "node_name", "container_name"},
		nil)
	mServicesDesc = prom.NewDesc(
		prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "services"),
		"The current information about service",
		[]string{"service_type", "node_id"},
		nil)
)

type Inventory struct {
	db             *reform.DB
	agentsRegistry *agents.Registry
}

func NewInventory(db *reform.DB, agentsRegistry *agents.Registry) *Inventory {
	i := &Inventory{
		db:             db,
		agentsRegistry: agentsRegistry,
	}
	return i
}

func (i *Inventory) Describe(chan<- *prom.Desc) {}

func (i *Inventory) Collect(ch chan<- prom.Metric) {
	ctx, cancelCtx := context.WithTimeout(context.Background(), cancelTime)
	defer cancelCtx()

	//l := logger.Get(ctx)

	var resAgents []*models.Agent
	var resNodes []*models.Node
	var resServices []*models.Service
	err := i.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		dbAgents, dbAgentsError := models.FindAgents(tx.Querier, models.AgentFilters{})
		dbNodes, dbNodesError := models.FindNodes(tx.Querier, models.NodeFilters{})
		dbServices, dbServicesError := models.FindServices(tx.Querier, models.ServiceFilters{})

		if dbAgentsError != nil {
			return dbAgentsError
		}

		if dbNodesError != nil {
			return dbNodesError
		}

		if dbServicesError != nil {
			return dbServicesError
		}

		resAgents = dbAgents
		resNodes = dbNodes
		resServices = dbServices

		return nil
	})

	if err != nil {
		fmt.Println(err)
		//l.Errorf("Failed with error %s", err)
	}

	for _, agent := range resAgents {
		var disabled = 0
		var connected float64 = 0

		pmmAgentId := pointer.GetString(agent.PMMAgentID)

		if agent.Disabled {
			disabled = 1
		} else {
			disabled = 0
		}

		if i.agentsRegistry.IsConnected(pmmAgentId) {
			connected = 1
		} else {
			connected = 0
		}

		agentMetricLabels := []string{
			string(agent.AgentType),
			pointer.GetString(agent.ServiceID),
			pointer.GetString(agent.NodeID),
			pmmAgentId,
			strconv.Itoa(disabled),
			pointer.GetString(agent.Version),
		}
		ch <- prom.MustNewConstMetric(mAgentsDesc, prom.GaugeValue, connected, agentMetricLabels...)
	}

	for _, node := range resNodes {
		nodeMetricLabels := []string{
			string(node.NodeType),
			node.NodeName,
			pointer.GetString(node.ContainerName),
		}
		ch <- prom.MustNewConstMetric(mNodesDesc, prom.GaugeValue, serviceEnabled, nodeMetricLabels...)
	}

	for _, service := range resServices {
		serviceMetricLabels := []string{
			string(service.ServiceType),
			service.NodeID,
		}
		ch <- prom.MustNewConstMetric(mServicesDesc, prom.GaugeValue, serviceEnabled, serviceMetricLabels...)
	}
}

var (
	_ prom.Collector = (*Inventory)(nil)
)

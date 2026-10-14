import { buildTableUrlParamKeys } from '@percona/peak-ui';
import {
  PMM_INVENTORY_PATH,
  PMM_INVENTORY_SERVICES_PATH,
  PMM_NEW_NAV_GRAFANA_PATH,
  PMM_QAN_URL,
} from 'lib/constants';
import { type ManagedAgent, AgentStatus, AgentType } from 'types/agent.types';
import { type ManagedService, ServiceStatus } from 'types/services.types';
import {
  INVENTORY_URL_PREFIX,
  LEGACY_ALL_COLUMNS,
  LEGACY_FILTER_PARAMS,
  LEGACY_FILTER_VALUES,
  LEGACY_SEARCH_COLUMN_PARAM,
  LEGACY_SEARCH_TEXT_PARAM,
  QAN_SERVICE_TYPES,
  SERVICE_DASHBOARDS,
} from './Inventory.constants';
import { Messages } from './Inventory.messages';
import {
  type InventoryEntity,
  type InventoryTab,
  MonitoringStatus,
} from './Inventory.types';

export const isAgentRunning = (agent: ManagedAgent) =>
  agent.status === AgentStatus.running ||
  agent.status === AgentStatus.starting ||
  agent.isConnected;

export const getAgentsMonitoringStatus = (agents: ManagedAgent[]) => {
  const allAgentsOk = agents.every(
    (agent) =>
      isAgentRunning(agent) ||
      (agent.disabled && agent.status === AgentStatus.done)
  );

  if (!allAgentsOk) {
    return MonitoringStatus.failed;
  }

  const disabledAgents = agents.filter(
    (agent) => agent.disabled && agent.agentType !== AgentType.rtaMongodbAgent
  );

  // pg_stat_statements is disabled on purpose when pg_stat_monitor is used
  if (
    disabledAgents.length > 1 ||
    (disabledAgents.length === 1 &&
      disabledAgents[0].agentType !== AgentType.qanPostgresqlPgstatementsAgent)
  ) {
    return MonitoringStatus.warning;
  }

  return MonitoringStatus.ok;
};

export const getStatusText = (status: ServiceStatus) =>
  Messages.status[status] ?? Messages.status[ServiceStatus.unspecified];

export const getStatusColor = (status: ServiceStatus) => {
  switch (status) {
    case ServiceStatus.up:
      return 'success';
    case ServiceStatus.down:
      return 'error';
    default:
      return 'default';
  }
};

export const getAgentsUrl = (tab: InventoryTab, id: string) =>
  `${PMM_INVENTORY_PATH}/${tab}/${id}/agents`;

// Links to a tab filtered on one of its columns, e.g. a node's services
export const getFilteredTabUrl = (
  tab: InventoryTab,
  columnId: string,
  value: string
) => {
  const { filterPrefix } = buildTableUrlParamKeys(INVENTORY_URL_PREFIX[tab]);
  const params = new URLSearchParams({ [`${filterPrefix}${columnId}`]: value });

  return `${PMM_INVENTORY_PATH}/${tab}?${params}`;
};

export const getServiceDashboardUrl = ({
  serviceType,
  serviceName,
}: ManagedService) => {
  const dashboard = SERVICE_DASHBOARDS[serviceType];

  return (
    dashboard &&
    `${PMM_NEW_NAV_GRAFANA_PATH}/d/${dashboard}?var-service_name=${encodeURIComponent(serviceName)}`
  );
};

export const getServiceQanUrl = ({
  serviceType,
  serviceName,
}: ManagedService) =>
  QAN_SERVICE_TYPES.includes(serviceType)
    ? `${PMM_QAN_URL}?var-service_name=${encodeURIComponent(serviceName)}`
    : undefined;

export const getEditServiceUrl = (serviceId: string) =>
  `${PMM_INVENTORY_SERVICES_PATH}/${serviceId}/edit`;

export const toLabelTags = (labels: Record<string, string>) =>
  Object.entries(labels)
    .filter(([, value]) => value !== '')
    .map(([key, value]) => `${key}=${value}`);

const getEntity = (segments: string[]): InventoryEntity => {
  if (segments[2] === 'agents') {
    return 'agents';
  }

  return segments[0] === 'nodes' ? 'nodes' : 'services';
};

export const translateLegacyParams = (search: string, prefix: string) => {
  const legacyParams = new URLSearchParams(search);
  const keys = buildTableUrlParamKeys(prefix);
  const params = new URLSearchParams();

  for (const [key, value] of legacyParams) {
    if (LEGACY_FILTER_PARAMS.includes(key)) {
      params.set(
        `${keys.filterPrefix}${key}`,
        LEGACY_FILTER_VALUES[value] ?? value
      );
    } else if (key.startsWith(`${prefix}.`)) {
      params.append(key, value);
    }
  }

  const text = legacyParams.get(LEGACY_SEARCH_TEXT_PARAM);
  const column = legacyParams.get(LEGACY_SEARCH_COLUMN_PARAM);

  if (text) {
    params.set(
      column && column !== LEGACY_ALL_COLUMNS
        ? `${keys.filterPrefix}${column}`
        : keys.globalFilter,
      text
    );
  }

  return params;
};

// Maps a former Grafana inventory URL (path below /graph/inventory) to the
// native page, translating its filters.
export const getInventoryRedirectUrl = (path: string, search: string) => {
  const segments = path.split('/').filter(Boolean);
  const pathname = segments.length
    ? `${PMM_INVENTORY_PATH}/${segments.join('/')}`
    : PMM_INVENTORY_SERVICES_PATH;
  const params = translateLegacyParams(
    search,
    INVENTORY_URL_PREFIX[getEntity(segments)]
  );

  const query = params.toString();

  return query ? `${pathname}?${query}` : pathname;
};

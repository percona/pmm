import type { ManagedAgent } from 'types/agent.types';
import {
  collectCsvColumns,
  type CsvRow,
  downloadCsv,
  toCsvValue,
} from 'utils/csv.utils';
import {
  getAgentStatusText,
  getAgentTypeName,
} from '../../agents/Agents.utils';
import { getStatusText, isAgentFailed } from '../../Inventory.utils';
import type { ServiceRow } from '../Services.types';

const SERVICES_EXPORT_FILENAME_PREFIX = 'inventory_services_export';

// Named as the API names the fields, like the RTA export. The table's columns
// come first; custom labels and other attributes follow alphabetically.
const SERVICES_CSV_COLUMNS = [
  'service_id',
  'service_name',
  'service_type',
  'status',
  'monitoring',
  'failed_agents',
  'node_id',
  'node_name',
  'address',
  'port',
  'environment',
  'cluster',
  'replication_set',
];

// The status is the only failure detail the API reports for an agent
const formatFailedAgent = ({ agentType, agentId, status }: ManagedAgent) =>
  `${getAgentTypeName(agentType)} (${agentId}): ${getAgentStatusText(status)}`;

export const mapServiceToCsvRow = (service: ServiceRow): CsvRow => {
  const fields: Record<string, unknown> = {
    service_id: service.serviceId,
    service_name: service.serviceName,
    service_type: service.serviceType,
    status: getStatusText(service.status),
    monitoring: service.agentsStatus,
    failed_agents: service.agents
      .filter(isAgentFailed)
      .map(formatFailedAgent)
      .join('; '),
    node_id: service.nodeId,
    node_name: service.nodeName,
    address: service.address,
    port: service.port,
  };
  // Custom labels and the service's other attributes, keyed as the API names
  // them. Label names are letters, digits and underscores, so a custom label
  // named like a field above keeps its value under a header no label can take.
  const fieldNames = Object.keys(fields);
  const labels = Object.entries(service.labels).map(([key, value]) => [
    fieldNames.includes(key) ? `label:${key}` : key,
    value,
  ]);

  return Object.fromEntries(
    [...Object.entries(fields), ...labels].map(([key, value]) => [
      key,
      toCsvValue(value),
    ])
  );
};

// Services carry different labels, so sorting them keeps the layout independent
// of which services are exported and in which order
const getColumnOrder = (rows: CsvRow[]) => [
  ...SERVICES_CSV_COLUMNS,
  ...collectCsvColumns(rows)
    .filter((column) => !SERVICES_CSV_COLUMNS.includes(column))
    .sort((a, b) => a.localeCompare(b)),
];

export const exportServicesToCsv = (services: ServiceRow[]): void => {
  const rows = services.map(mapServiceToCsvRow);

  downloadCsv(rows, SERVICES_EXPORT_FILENAME_PREFIX, getColumnOrder(rows));
};

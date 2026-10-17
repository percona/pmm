import { Chip } from '@percona/peak-ui';
import { type MRT_ColumnDef } from 'material-react-table';
import { type ManagedAgent, AgentStatus } from 'types/agent.types';
import { Messages } from './Agents.messages';
import {
  getAgentStatusColor,
  getAgentStatusText,
  getAgentTypeName,
} from './Agents.utils';

const AGENT_TYPE_OPTIONS = Object.entries(Messages.agentTypes)
  .map(([value, label]) => ({ value, label }))
  .sort((a, b) => a.label.localeCompare(b.label));

const AGENT_STATUS_OPTIONS = [
  AgentStatus.done,
  AgentStatus.initializationError,
  AgentStatus.running,
  AgentStatus.starting,
  AgentStatus.stopping,
  AgentStatus.unknown,
  AgentStatus.waiting,
].map((value) => ({ value, label: getAgentStatusText(value) }));

export const AGENTS_COLUMNS: MRT_ColumnDef<ManagedAgent>[] = [
  {
    accessorKey: 'status',
    header: Messages.columns.status,
    Cell: ({ row: { original } }) => (
      <Chip
        size="small"
        label={getAgentStatusText(original.status)}
        color={getAgentStatusColor(original.status)}
        data-testid="agent-status-chip"
      />
    ),
    filterVariant: 'select',
    filterSelectOptions: AGENT_STATUS_OPTIONS,
    filterFn: 'equals',
    enableGlobalFilter: false,
  },
  {
    // Named as in the Grafana page so its links keep filtering
    id: 'type',
    accessorKey: 'agentType',
    header: Messages.columns.agentType,
    Cell: ({ row: { original } }) => getAgentTypeName(original.agentType),
    filterVariant: 'select',
    filterSelectOptions: AGENT_TYPE_OPTIONS,
    filterFn: 'equals',
    enableGlobalFilter: false,
  },
  {
    accessorKey: 'agentId',
    header: Messages.columns.agentId,
    filterFn: 'contains',
  },
];

import { type MRT_ColumnDef } from 'material-react-table';
import { AgentStatus } from 'types/agent.types';
import { RealTimeSession, RunningRealTimeAgent } from 'types/rta.types';

export const SESSIONS_TABLE_COLUMNS: MRT_ColumnDef<RealTimeSession>[] = [
  {
    accessorKey: 'sessionName',
    header: 'Session',
  },
  {
    accessorKey: 'status',
    header: 'Status',
  },
];

export const MOCK_DATA: RunningRealTimeAgent[] = [
  {
    agentId: '1',
    serviceId: '1',
    serviceName: 'service1',
    startedAt: new Date(),
    cluster: 'cluster1',
    status: AgentStatus.RUNNING,
  },
  {
    agentId: '2',
    serviceId: '2',
    serviceName: 'service2',
    startedAt: new Date(),
    cluster: 'cluster2',
    status: AgentStatus.WAITING,
  },
  {
    agentId: '3',
    serviceId: '3',
    serviceName: 'service3',
    startedAt: new Date(),
    cluster: 'cluster3',
    status: AgentStatus.STOPPING,
  },
  {
    agentId: '4',
    serviceId: '4',
    serviceName: 'service4',
    startedAt: new Date(),
    cluster: '',
    status: AgentStatus.DONE,
  },
  {
    agentId: '5',
    serviceId: '5',
    serviceName: 'service5',
    startedAt: new Date(),
    cluster: '',
    status: AgentStatus.UNKNOWN,
  },
  {
    agentId: '6',
    serviceId: '6',
    serviceName: 'service6',
    startedAt: new Date(),
    cluster: '',
    status: AgentStatus.STARTING,
  },
];

import { AgentUpdateSeverity, GetAgentVersionItem } from 'types/agent.types';
import { VersionsFilter } from './UpdateClients.types';

export const filterClients = (
  clients: GetAgentVersionItem[],
  filter: VersionsFilter
) => clients.filter((client) => shouldShowClient(client, filter));

const shouldShowClient = (
  client: GetAgentVersionItem,
  filter: VersionsFilter
) =>
  filter === VersionsFilter.All ||
  (filter === VersionsFilter.Critical &&
    client.severity === AgentUpdateSeverity.CRITICAL) ||
  (filter === VersionsFilter.Required &&
    client.severity === AgentUpdateSeverity.REQUIRED);

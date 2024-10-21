import { AgentUpdateSeverity, GetAgentVersionItem } from 'types/agent.types';
import { UpdateStatus } from 'types/updates.types';

export const areClientsUpToDate = (clients?: GetAgentVersionItem[]) =>
  !!clients?.every(
    (client) => client.severity === AgentUpdateSeverity.UP_TO_DATE
  );

export const isUpdateInProgress = (status: UpdateStatus) =>
  status === UpdateStatus.Updating ||
  status === UpdateStatus.Restarting ||
  status === UpdateStatus.Completed;

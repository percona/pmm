import type { GetAgentVersionItem } from 'types/agent.types';
import type { GetUpdatesResponse, UpdateStatus } from 'types/updates.types';

export interface UpdatesContextProps {
  isLoading: boolean;
  inProgress: boolean;
  status: UpdateStatus;
  setStatus: (status: UpdateStatus) => void;
  versionInfo?: GetUpdatesResponse;
  recheck: () => void;
  clients?: GetAgentVersionItem[];
  areClientsUpToDate: boolean;
}

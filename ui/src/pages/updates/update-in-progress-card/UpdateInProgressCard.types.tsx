import { CurrentInfo, LatestInfo, UpdateStatus } from 'types/updates.types';

export interface UpdateInProgressCardProps {
  versionInfo: Partial<CurrentInfo & LatestInfo>;
  status: UpdateStatus;
  authToken?: string;
}

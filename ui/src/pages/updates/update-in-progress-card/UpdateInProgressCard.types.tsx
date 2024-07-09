import { CurrentInfo, LatestInfo, UpdateStatus } from 'types/updates.types';

export interface UpdateInProgressCardProps {
  versionInfo: CurrentInfo & LatestInfo;
  status: UpdateStatus;
}

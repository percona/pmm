import { UpdateStatus, VersionInfo } from 'types/updates.types';

export interface UpdateInProgressCardProps {
  versionInfo: VersionInfo;
  status: UpdateStatus;
}

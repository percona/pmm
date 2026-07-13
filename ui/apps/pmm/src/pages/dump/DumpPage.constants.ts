import { DumpStatus } from 'types/dump.types';
import { DumpStatusColor } from './DumpPage.types';

export const DOWNLOAD_DELAY_MS = 900;
export const LOG_CHUNK_LIMIT = 200;
export const LOG_REFETCH_INTERVAL_MS = 3_000;

export const STATUS_COLORS: Record<DumpStatus, DumpStatusColor> = {
  [DumpStatus.Unspecified]: 'default',
  [DumpStatus.InProgress]: 'info',
  [DumpStatus.Success]: 'success',
  [DumpStatus.Error]: 'error',
};

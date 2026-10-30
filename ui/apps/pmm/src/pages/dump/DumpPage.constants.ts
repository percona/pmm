import { ChipProps } from '@mui/material/Chip';
import { DumpStatus } from 'types/dump.types';
import { Messages } from './DumpPage.messages';

export const DOWNLOAD_DELAY_MS = 900;
export const LOG_CHUNK_LIMIT = 200;
export const LOG_REFETCH_INTERVAL_MS = 3_000;

export const DUMP_STATUS: Record<
  DumpStatus,
  { label: string; color: ChipProps['color'] }
> = {
  [DumpStatus.Unspecified]: {
    label: Messages.status.unspecified,
    color: 'default',
  },
  [DumpStatus.InProgress]: { label: Messages.status.inProgress, color: 'info' },
  [DumpStatus.Success]: { label: Messages.status.success, color: 'success' },
  [DumpStatus.Error]: { label: Messages.status.error, color: 'error' },
};

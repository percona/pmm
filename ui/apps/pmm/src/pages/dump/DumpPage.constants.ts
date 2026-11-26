import type { StatusIconSeverity } from '@percona/peak-ui';
import type { UseTableUrlStateOptions } from 'hooks/utils/useTableUrlState';
import { DumpStatus } from 'types/dump.types';
import { Messages } from './DumpPage.messages';

export const DOWNLOAD_DELAY_MS = 900;
export const LOG_CHUNK_LIMIT = 200;
export const LOG_REFETCH_INTERVAL_MS = 3_000;
export const MAX_VISIBLE_SERVICES = 2;

export const DUMP_TABLE_NAME = 'pmm-dump';

export const DUMP_URL_STATE_OPTIONS: UseTableUrlStateOptions = {
  paramPrefix: 'dumps',
  defaults: {
    sorting: [{ id: 'createdAt', desc: true }],
    pagination: { pageIndex: 0, pageSize: 25 },
  },
};

export const PMM_DUMP_DOCS_URL =
  'https://docs.percona.com/percona-monitoring-and-management/3/troubleshoot/pmm_dump.html';

export const DUMP_STATUS: Record<
  DumpStatus,
  { label: string; severity: StatusIconSeverity }
> = {
  [DumpStatus.Unspecified]: {
    label: Messages.status.unknown,
    severity: 'indeterminate',
  },
  [DumpStatus.InProgress]: {
    label: Messages.status.creating,
    severity: 'info',
  },
  [DumpStatus.Success]: { label: Messages.status.ready, severity: 'success' },
  [DumpStatus.Error]: { label: Messages.status.failed, severity: 'error' },
};

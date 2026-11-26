import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { StatusIcon } from '@percona/peak-ui';
import type { FC } from 'react';
import { type Dump, DumpStatus as Status } from 'types/dump.types';
import { DUMP_STATUS } from '../../DumpPage.constants';
import { Messages } from '../../DumpPage.messages';
import { formatRelativeTime, isDumpCreating } from '../../DumpPage.utils';

interface Props {
  dump: Dump;
  onShowLogs: () => void;
}

export const DumpStatus: FC<Props> = ({ dump, onShowLogs }) => {
  const { label, severity } =
    DUMP_STATUS[dump.status] ?? DUMP_STATUS[Status.Unspecified];

  return (
    <Stack direction="row" gap={1} alignItems="flex-start">
      <StatusIcon severity={severity} size="small" />
      <Stack alignItems="flex-start">
        <Typography variant="body2">{label}</Typography>
        {isDumpCreating(dump) && (
          <Typography variant="caption" color="text.secondary">
            {Messages.status.startedAgo(formatRelativeTime(dump.createdAt))}
          </Typography>
        )}
        {dump.status === Status.Error && (
          <Link
            component="button"
            variant="caption"
            onClick={onShowLogs}
            data-testid={`dump-status-logs-${dump.dumpId}`}
          >
            {Messages.status.seeWhatWentWrong}
          </Link>
        )}
      </Stack>
    </Stack>
  );
};

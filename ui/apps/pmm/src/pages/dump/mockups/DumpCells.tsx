import LockOutlinedIcon from '@mui/icons-material/LockOutlined';
import CircularProgress from '@mui/material/CircularProgress';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Chip, StatusIcon, Tooltip } from '@percona/peak-ui';
import { format, formatDistanceStrict, formatDistanceToNow } from 'date-fns';
import type { FC } from 'react';
import { type Dump, DumpStatus } from 'types/dump.types';

const VISIBLE_SERVICES = 2;

export const formatWindow = ({ startTime, endTime }: Dump) => {
  const start = new Date(startTime);
  const end = new Date(endTime);
  const sameDay = start.toDateString() === end.toDateString();

  return `${format(start, 'MMM d, HH:mm')} – ${format(end, sameDay ? 'HH:mm' : 'MMM d, HH:mm')}`;
};

export const formatDuration = ({ startTime, endTime }: Dump) =>
  formatDistanceStrict(new Date(endTime), new Date(startTime));

export const DumpStatusCell: FC<{ dump: Dump; onViewLogs: () => void }> = ({
  dump,
  onViewLogs,
}) => {
  if (dump.status === DumpStatus.InProgress) {
    return (
      <Stack direction="row" gap={1} alignItems="center">
        <CircularProgress size={18} aria-hidden />
        <Stack>
          <Typography variant="body2">Creating…</Typography>
          <Typography variant="caption" color="text.secondary">
            Started {formatDistanceToNow(new Date(dump.createdAt))} ago
          </Typography>
        </Stack>
      </Stack>
    );
  }

  if (dump.status === DumpStatus.Error) {
    return (
      <Stack direction="row" gap={1} alignItems="center">
        <StatusIcon severity="error" size="small" />
        <Stack>
          <Typography variant="body2">Failed</Typography>
          <Link
            component="button"
            variant="caption"
            onClick={onViewLogs}
            sx={{ textAlign: 'left' }}
          >
            See what went wrong
          </Link>
        </Stack>
      </Stack>
    );
  }

  return (
    <Stack direction="row" gap={1} alignItems="center">
      <StatusIcon
        severity={
          dump.status === DumpStatus.Success ? 'success' : 'indeterminate'
        }
        size="small"
      />
      <Typography variant="body2">
        {dump.status === DumpStatus.Success ? 'Ready' : 'Unknown'}
      </Typography>
    </Stack>
  );
};

export const ServicesCell: FC<{ services: string[] }> = ({ services }) => {
  if (!services.length) {
    return <Chip size="small" variant="outlined" label="All services" />;
  }

  const hidden = services.slice(VISIBLE_SERVICES);

  return (
    <Stack direction="row" gap={0.5} flexWrap="wrap">
      {services.slice(0, VISIBLE_SERVICES).map((service) => (
        <Chip key={service} size="small" label={service} />
      ))}
      {!!hidden.length && (
        <Tooltip title={hidden.join(', ')}>
          <Chip size="small" variant="outlined" label={`+${hidden.length}`} />
        </Tooltip>
      )}
    </Stack>
  );
};

export const CreatedCell: FC<{ dump: Dump }> = ({ dump }) => (
  <Tooltip title={format(new Date(dump.createdAt), 'yyyy-MM-dd HH:mm:ss')}>
    <Typography variant="body2">
      {formatDistanceToNow(new Date(dump.createdAt), { addSuffix: true })}
    </Typography>
  </Tooltip>
);

export const WindowCell: FC<{ dump: Dump }> = ({ dump }) => (
  <Stack>
    <Typography variant="body2">{formatWindow(dump)}</Typography>
    <Stack direction="row" gap={0.5} alignItems="center">
      <Typography variant="caption" color="text.secondary">
        {formatDuration(dump)}
      </Typography>
      {dump.encrypted && (
        <Tooltip title="Encrypted. Opening it needs the password set when it was created.">
          <Stack direction="row" gap={0.25} alignItems="center">
            <LockOutlinedIcon sx={{ fontSize: 14 }} color="action" />
            <Typography variant="caption" color="text.secondary">
              Encrypted
            </Typography>
          </Stack>
        </Tooltip>
      )}
    </Stack>
  </Stack>
);

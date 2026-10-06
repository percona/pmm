/**
 * What the Backups tab should answer first: which backups this cluster actually has.
 *
 * The tab opened on a table of SEP *tasks* -- a record of what someone asked for,
 * which is not the same question as what exists to restore from. A task that ran
 * successfully a month ago says nothing about whether its backup is still in the
 * bucket, and PBM's own retention can remove one without any task knowing.
 *
 * PMM already scrapes the answer. Three gauges carry one series per backup, keyed by
 * PBM's backup name -- which is also the handle a restore takes -- with its type,
 * status, size and completion time as labels. That is `pbm list`, free, with no
 * dispatch, no credential and no SEP in the path.
 *
 * The task table stays below as history, the same division the Configuration tab
 * makes: state on top, the record of attempts underneath.
 */
import { Box, Chip, Skeleton, Stack, Typography } from '@mui/material';
import { useQuery } from '@tanstack/react-query';

import { fetchPbmConfigStatus, type PbmBackup } from './pbmConfigStatus';
import type { PbmCluster } from './pbmClusters';

/** Matches the switcher's cadence: PBM's own view changes slowly. */
const REFETCH_MS = 30_000;

/** Bytes as a short human string; PBM reports exact sizes and nobody reads those. */
function humanBytes(bytes?: number): string {
  if (bytes === undefined) {
    return '—';
  }
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB'];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  return `${value < 10 && unit > 0 ? value.toFixed(1) : Math.round(value)} ${units[unit]}`;
}

/** Seconds as a short duration. */
function humanDuration(seconds?: number): string {
  if (seconds === undefined) {
    return '—';
  }
  if (seconds < 60) {
    return `${Math.round(seconds)}s`;
  }
  if (seconds < 3600) {
    return `${Math.round(seconds / 60)}m`;
  }
  return `${(seconds / 3600).toFixed(1)}h`;
}

/**
 * Render a backup's completion as a local time.
 *
 * PBM's own name for a backup is its *start* in UTC, so showing that alone would
 * leave a reader converting timezones to answer "how old is this".
 */
function completedAt(epochSeconds?: number): string {
  if (epochSeconds === undefined) {
    return '—';
  }
  return new Date(epochSeconds * 1000).toLocaleString();
}

/** PBM's status vocabulary, mapped onto the chip colours PMM uses elsewhere. */
function statusColor(status: string): 'success' | 'error' | 'info' | 'default' {
  if (status === 'done') {
    return 'success';
  }
  if (status === 'error' || status === 'canceled') {
    return 'error';
  }
  if (status === 'running' || status === 'starting') {
    return 'info';
  }
  return 'default';
}

function BackupRow({ backup }: { backup: PbmBackup }) {
  return (
    <Stack
      direction="row"
      spacing={2}
      alignItems="center"
      flexWrap="wrap"
      useFlexGap
    >
      {/*
        PBM's backup name is the restore handle, so it is shown verbatim and never
        reformatted -- someone copying it into `pbm restore` must get what PBM has.
      */}
      <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
        {backup.name}
      </Typography>
      <Chip size="small" variant="outlined" label={backup.type || 'unknown'} />
      <Chip
        size="small"
        label={backup.status || 'unknown'}
        color={statusColor(backup.status)}
      />
      <Typography variant="body2" color="text.secondary">
        {humanBytes(backup.sizeBytes)} · took{' '}
        {humanDuration(backup.durationSeconds)} ·{' '}
        {completedAt(backup.lastTransition)}
      </Typography>
    </Stack>
  );
}

/**
 * Render the backups PBM holds for one cluster.
 *
 * @param cluster The selected cluster, or undefined while topology is loading.
 */
export function PbmBackupsPanel({ cluster }: { cluster?: PbmCluster }) {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['backup-mongo:pbm-config-status', cluster?.name],
    queryFn: () => fetchPbmConfigStatus(cluster?.name ?? ''),
    enabled: Boolean(cluster),
    refetchInterval: REFETCH_MS,
  });

  if (!cluster) {
    return null;
  }
  if (isLoading) {
    return <Skeleton variant="rounded" height={96} sx={{ mb: 3 }} />;
  }
  if (isError) {
    return (
      <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
        Could not read PBM state from PMM.
      </Typography>
    );
  }

  const backups = data?.backups ?? [];

  return (
    <Box sx={{ mb: 3 }}>
      <Stack
        direction="row"
        spacing={1}
        alignItems="baseline"
        sx={{ mb: backups.length > 0 ? 1.5 : 0 }}
      >
        <Typography variant="subtitle2">Backups</Typography>
        <Typography variant="caption" color="text.secondary">
          {cluster.name}
          {backups.length > 0 && ` · ${backups.length} held by PBM`}
        </Typography>
      </Stack>

      {backups.length === 0 ? (
        <Typography variant="body2" color="text.secondary">
          PBM holds no backups for this cluster.
        </Typography>
      ) : (
        <Stack spacing={1}>
          {backups.map((backup) => (
            <BackupRow key={backup.name} backup={backup} />
          ))}
        </Stack>
      )}
    </Box>
  );
}

/**
 * What the Configuration tab should say first: this cluster's PBM state.
 *
 * The tab used to open on a table of configuration *tasks* -- a log of attempts to
 * configure, not what the cluster is configured with. That is the framework's
 * generic surface for a task app, right for Backups and Restores and the wrong
 * question here. This panel answers it, from metrics PMM already scrapes, with no
 * dispatch and no credential.
 *
 * It deliberately does not claim to show the config document. Storage type, bucket,
 * retention and priorities are not in any metric, and saying "configured" is as far
 * as this data honestly goes -- see `pbmConfigStatus.ts`.
 */
import { Box, Chip, Skeleton, Stack, Typography } from '@mui/material';
import {
  ClusterHealthyIcon,
  ClusterInoperationalIcon,
  Tooltip,
} from '@percona/peak-ui';
import { useQuery } from '@tanstack/react-query';

import { fetchPbmConfigStatus, type PbmBackup } from './pbmConfigStatus';
import type { PbmCluster } from './pbmClusters';

/** Matches the switcher's cadence: PBM's own view changes slowly. */
const REFETCH_MS = 30_000;

function humanBytes(bytes?: number) {
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

function humanAge(epochSeconds?: number) {
  if (!epochSeconds) {
    return '—';
  }
  const seconds = Math.max(0, Date.now() / 1000 - epochSeconds);
  const days = Math.floor(seconds / 86400);
  if (days > 0) {
    return `${days}d ago`;
  }
  const hours = Math.floor(seconds / 3600);
  if (hours > 0) {
    return `${hours}h ago`;
  }
  return `${Math.max(1, Math.floor(seconds / 60))}m ago`;
}

function BackupRow({ backup }: { backup: PbmBackup }) {
  return (
    <Stack
      direction="row"
      spacing={1.5}
      alignItems="center"
      sx={{ fontSize: 14 }}
    >
      <Chip
        size="small"
        label={backup.status || 'unknown'}
        color={
          backup.status === 'done'
            ? 'success'
            : backup.status === 'error'
              ? 'error'
              : 'default'
        }
      />
      <Typography variant="body2" sx={{ minWidth: 190 }}>
        {backup.name}
      </Typography>
      <Typography variant="body2" color="text.secondary">
        {backup.type || '—'}
      </Typography>
      <Typography variant="body2" color="text.secondary">
        {humanBytes(backup.sizeBytes)}
      </Typography>
      <Typography variant="body2" color="text.secondary">
        {humanAge(backup.lastTransition)}
      </Typography>
    </Stack>
  );
}

/**
 * Render the status panel for one cluster.
 *
 * @param cluster The selected cluster, or undefined while topology is loading.
 */
export function PbmConfigStatusPanel({ cluster }: { cluster?: PbmCluster }) {
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

  const live = cluster.members.filter((m) => m.healthy).length;
  const configured = data?.storageConfigured;

  return (
    <Box sx={{ mb: 3 }}>
      <Stack
        direction="row"
        spacing={2}
        alignItems="center"
        flexWrap="wrap"
        sx={{ mb: 1.5 }}
      >
        <Tooltip title={`${live} of ${cluster.members.length} agents healthy`}>
          {cluster.allHealthy ? (
            <ClusterHealthyIcon fontSize="small" />
          ) : (
            <ClusterInoperationalIcon fontSize="small" />
          )}
        </Tooltip>
        <Typography variant="subtitle2">{cluster.name}</Typography>
        <Chip
          size="small"
          label={
            configured === undefined
              ? 'storage unknown'
              : configured
                ? 'storage configured'
                : 'no storage configured'
          }
          color={
            configured
              ? 'success'
              : configured === false
                ? 'warning'
                : 'default'
          }
        />
        <Chip
          size="small"
          label={data?.pitrEnabled ? 'PITR on' : 'PITR off'}
          color={data?.pitrEnabled ? 'success' : 'default'}
        />
        <Typography variant="body2" color="text.secondary">
          {live}/{cluster.members.length} agents
          {cluster.sharded && ` · ${cluster.replicaSets.length} replica sets`}
        </Typography>
      </Stack>

      {data?.backups.length ? (
        <Stack spacing={0.75}>
          <Typography variant="caption" color="text.secondary">
            Recent backups
          </Typography>
          {data.backups.slice(0, 5).map((backup) => (
            <BackupRow key={backup.name} backup={backup} />
          ))}
        </Stack>
      ) : (
        <Typography variant="body2" color="text.secondary">
          No backups recorded for this cluster yet.
        </Typography>
      )}
    </Box>
  );
}

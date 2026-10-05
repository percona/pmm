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

import { fetchPbmConfigStatus } from './pbmConfigStatus';
import type { PbmCluster } from './pbmClusters';

/** Matches the switcher's cadence: PBM's own view changes slowly. */
const REFETCH_MS = 30_000;

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
    </Box>
  );
}

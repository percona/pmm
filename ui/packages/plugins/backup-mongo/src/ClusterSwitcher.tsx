/**
 * Top-level cluster switcher for MongoDB Backups.
 *
 * Sits above the tabs because the cluster scopes all three of them: configuration,
 * backups and restores are each about *a* deployment, and PBM's own configuration is
 * cluster-wide rather than per-node.
 *
 * The choice lives in the URL rather than in component state, for the reason the tab
 * bar keeps its own selection there: a bookmarked or reloaded page must come back to
 * the same view, and a cluster someone is midway through configuring is exactly what
 * they would bookmark.
 *
 * Peak supplies the vocabulary -- the toggle button, the cluster health icons and the
 * tooltip -- while MUI supplies the grouping primitive Peak builds on
 * (`ToggleRegularButton` is a styled `ToggleButton`). That mix is what the PMM app
 * itself does. Peak's `ToggleButtonGroupInput` and `SelectInput` are react-hook-form
 * fields taking `name` and `control`; this switcher is URL-driven and belongs to no
 * form, so the plain components are the right ones.
 */
import { useMemo } from 'react';

import {
  Box,
  MenuItem,
  Skeleton,
  TextField,
  ToggleButtonGroup,
  Typography,
} from '@mui/material';
import {
  ClusterHealthyIcon,
  ClusterInoperationalIcon,
  ToggleRegularButton,
  Tooltip,
} from '@percona/peak-ui';
import { useQuery } from '@tanstack/react-query';
import { useSearchParams } from 'react-router-dom';

import { fetchPbmClusters, type PbmCluster } from './pbmClusters';

/** The search param the selected cluster lives in. */
export const CLUSTER_PARAM = 'cluster';

/**
 * Above this many clusters a segmented control stops being readable and starts
 * wrapping; a select degrades gracefully where buttons do not. Chosen for legibility
 * rather than measured -- adjust it when a real estate says otherwise.
 */
const MAX_SEGMENTED = 4;

/**
 * How often to re-read PBM state.
 *
 * Deliberately unhurried. The query is cheap, but the *value* it returns is PBM's own
 * view, which times out a heartbeat before declaring an agent lost -- measured at
 * about 45s for `pbm status` to notice a stopped agent and about a minute more for the
 * metric to follow. Polling faster than the underlying state can change only adds load
 * and invites the reader to believe the icon is live.
 */
const REFETCH_MS = 30_000;

export const PBM_CLUSTERS_QUERY_KEY = ['backup-mongo:pbm-clusters'];

/** Summarise a cluster's agent health for a tooltip and for assistive technology. */
function healthLabel(cluster: PbmCluster) {
  const total = cluster.members.length;
  const healthy = cluster.members.filter((member) => member.healthy).length;
  return healthy === total
    ? `${total} PBM agent${total === 1 ? '' : 's'}, all healthy`
    : `${healthy} of ${total} PBM agents healthy`;
}

/**
 * Peak's own cluster health icons, which carry the meaning rather than relying on a
 * coloured dot -- the state is legible without colour vision.
 */
function ClusterHealth({ cluster }: { cluster: PbmCluster }) {
  const label = healthLabel(cluster);
  const Icon = cluster.allHealthy
    ? ClusterHealthyIcon
    : ClusterInoperationalIcon;
  return (
    <Tooltip title={label}>
      <Icon titleAccess={label} fontSize="small" />
    </Tooltip>
  );
}

export interface ClusterSwitcherProps {
  /** Called with the resolved selection, so the surrounding app can scope itself. */
  onChange?: (cluster: PbmCluster | undefined) => void;
}

/**
 * Render the switcher and report the current selection.
 *
 * @param onChange Notified whenever the resolved selection changes.
 */
export function ClusterSwitcher({ onChange }: ClusterSwitcherProps) {
  const [searchParams, setSearchParams] = useSearchParams();

  const { data, isLoading, isError } = useQuery({
    queryKey: PBM_CLUSTERS_QUERY_KEY,
    queryFn: fetchPbmClusters,
    refetchInterval: REFETCH_MS,
  });

  const clusters = useMemo(() => data ?? [], [data]);
  const requested = searchParams.get(CLUSTER_PARAM);
  // Fall back to the first cluster rather than to "none": the tabs below are useless
  // without one, and someone with a single cluster should never have to pick it.
  const selected =
    clusters.find((cluster) => cluster.name === requested) ?? clusters[0];

  useMemo(() => onChange?.(selected), [selected, onChange]);

  const select = (name: string) => {
    const next = new URLSearchParams(searchParams);
    next.set(CLUSTER_PARAM, name);
    // Replace rather than push: switching cluster changes the view, it does not
    // navigate, and it should not fill the back button with every toggle.
    setSearchParams(next, { replace: true });
  };

  if (isLoading) {
    return (
      <Skeleton variant="rounded" width={320} height={40} sx={{ mb: 2 }} />
    );
  }

  if (isError || clusters.length === 0) {
    return (
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        {isError
          ? 'Could not read PBM agent status from PMM.'
          : 'No MongoDB cluster is reporting a PBM agent to PMM.'}
      </Typography>
    );
  }

  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 2 }}>
      {clusters.length <= MAX_SEGMENTED ? (
        <ToggleButtonGroup
          exclusive
          size="small"
          value={selected?.name ?? null}
          onChange={(_event, name) => name && select(name)}
          aria-label="MongoDB cluster"
        >
          {clusters.map((cluster) => (
            <ToggleRegularButton
              key={cluster.name}
              value={cluster.name}
              dataTestId={`cluster-toggle-${cluster.name}`}
              sx={{ gap: 1 }}
            >
              <ClusterHealth cluster={cluster} />
              {cluster.name}
            </ToggleRegularButton>
          ))}
        </ToggleButtonGroup>
      ) : (
        <TextField
          select
          size="small"
          label="Cluster"
          value={selected?.name ?? ''}
          onChange={(event) => select(event.target.value)}
          sx={{ minWidth: 260 }}
        >
          {clusters.map((cluster) => (
            <MenuItem key={cluster.name} value={cluster.name}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <ClusterHealth cluster={cluster} />
                {cluster.name}
              </Box>
            </MenuItem>
          ))}
        </TextField>
      )}

      {selected && (
        <Typography variant="body2" color="text.secondary">
          {selected.members
            .map(
              (member) =>
                `${member.serviceName} ${member.role === 'P' ? 'PRIMARY' : 'SECONDARY'}`
            )
            .join(' · ')}
        </Typography>
      )}
    </Box>
  );
}

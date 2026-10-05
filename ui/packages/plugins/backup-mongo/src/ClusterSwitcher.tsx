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
 * **Why a search box and not a row of buttons.** A segmented toggle shows the whole
 * estate at once, which is pleasant at three clusters and unusable at three hundred --
 * it wraps into a wall, every cluster costs a button, and finding one means reading.
 * Typing is the only selection gesture whose cost does not grow with the estate, so
 * the switcher is a combobox at every size rather than changing shape at a threshold.
 * Two things make it work at the top end: degraded clusters are grouped first, because
 * in a large estate those are what someone is hunting for, and the rendered list is
 * capped (see `OPTION_LIMIT`) so the listbox stays bounded however many clusters
 * exist.
 *
 * Peak supplies the vocabulary -- the cluster health icons and the tooltip -- while
 * MUI supplies the primitive Peak builds on. Peak's `AutoCompleteInput` is a
 * react-hook-form field taking `name` and `control`; this switcher is URL-driven and
 * belongs to no form, and giving it a form purely to host one field would make the URL
 * and the form state two things to keep in sync. The plain component is the right one,
 * as it already was for the toggle this replaces.
 *
 * Topology -- who is primary, which member runs `pbm` -- is deliberately *not* here.
 * It is reference material for someone already looking at a cluster, not something to
 * read while choosing between clusters, so it lives in the Configuration tab's status
 * panel.
 */
import { useMemo } from 'react';

import {
  Autocomplete,
  Box,
  Skeleton,
  TextField,
  Typography,
  createFilterOptions,
} from '@mui/material';
import {
  ClusterHealthyIcon,
  ClusterInoperationalIcon,
  Tooltip,
} from '@percona/peak-ui';
import { useQuery } from '@tanstack/react-query';
import { useSearchParams } from 'react-router-dom';

import { fetchPbmClusters, type PbmCluster } from './pbmClusters';

/** The search param the selected cluster lives in. */
export const CLUSTER_PARAM = 'cluster';

/**
 * How many options to render at once.
 *
 * Not a limit on the estate -- every cluster remains reachable by typing, and the
 * filter runs over all of them. It bounds the *listbox*, which is what would otherwise
 * grow a DOM node per cluster on every open. Past a screenful nobody scrolls a
 * combobox anyway; they type.
 */
const OPTION_LIMIT = 50;

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

const filterClusters = createFilterOptions<PbmCluster>({
  limit: OPTION_LIMIT,
  stringify: (cluster) => cluster.name,
});

/** Summarise a cluster's agent health for a tooltip and for assistive technology. */
function healthLabel(cluster: PbmCluster) {
  const total = cluster.members.length;
  const healthy = cluster.members.filter((member) => member.healthy).length;
  return healthy === total
    ? `${total} PBM agent${total === 1 ? '' : 's'}, all healthy`
    : `${healthy} of ${total} PBM agents healthy`;
}

/** The one line under a cluster's name: shape first, then agent count. */
function shapeLabel(cluster: PbmCluster) {
  const shape = cluster.sharded
    ? `sharded · ${cluster.replicaSets.length} replica sets`
    : 'replica set';
  return `${shape} · ${healthLabel(cluster)}`;
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
  // without one, and someone with a single cluster should never have to pick it. The
  // fallback reads the name-ordered list, not the display order below, so a cluster
  // going degraded never silently moves the default selection.
  const selected =
    clusters.find((cluster) => cluster.name === requested) ?? clusters[0];

  // Degraded first. At three clusters this is cosmetic; at three hundred it is the
  // difference between finding the broken one and scrolling for it.
  const options = useMemo(
    () =>
      [...clusters].sort(
        (a, b) =>
          Number(a.allHealthy) - Number(b.allHealthy) ||
          a.name.localeCompare(b.name)
      ),
    [clusters]
  );
  const degraded = clusters.filter((cluster) => !cluster.allHealthy).length;

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
      <Skeleton variant="rounded" width={360} height={40} sx={{ mb: 2 }} />
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
      <Autocomplete
        options={options}
        value={selected ?? null}
        // There is always a selection; clearing it would leave the tabs below
        // scoped to nothing, which is not a state the page can render.
        disableClearable
        openOnFocus
        autoHighlight
        blurOnSelect
        size="small"
        sx={{ width: 360 }}
        filterOptions={filterClusters}
        getOptionLabel={(cluster) => cluster.name}
        isOptionEqualToValue={(option, value) => option.name === value.name}
        groupBy={(cluster) =>
          cluster.allHealthy ? 'Healthy' : 'Needs attention'
        }
        onChange={(_event, cluster) => cluster && select(cluster.name)}
        data-testid="cluster-select"
        renderInput={(params) => (
          <TextField
            {...params}
            label="Cluster"
            slotProps={{
              input: {
                ...params.InputProps,
                startAdornment: selected ? (
                  <Box sx={{ display: 'flex', pl: 0.5 }}>
                    <ClusterHealth cluster={selected} />
                  </Box>
                ) : null,
              },
            }}
          />
        )}
        renderOption={(props, cluster) => {
          const { key, ...rest } = props as typeof props & { key?: string };
          return (
            <Box
              component="li"
              key={key ?? cluster.name}
              {...rest}
              sx={{ gap: 1, alignItems: 'flex-start !important' }}
            >
              <Box sx={{ display: 'flex', pt: 0.25 }}>
                <ClusterHealth cluster={cluster} />
              </Box>
              <Box sx={{ minWidth: 0 }}>
                <Typography variant="body2" noWrap>
                  {cluster.name}
                </Typography>
                <Typography variant="caption" color="text.secondary" noWrap>
                  {shapeLabel(cluster)}
                </Typography>
              </Box>
            </Box>
          );
        }}
      />

      {/*
        The estate at a glance, so the search box is not the only clue to how much
        it is hiding -- and so a degraded cluster announces itself without the list
        being open.
      */}
      <Typography variant="body2" color="text.secondary">
        {clusters.length} cluster{clusters.length === 1 ? '' : 's'}
        {degraded > 0 && ` · ${degraded} need attention`}
      </Typography>
    </Box>
  );
}

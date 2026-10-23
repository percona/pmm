import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Paper from '@mui/material/Paper';
import {
  Chip,
  EmptyState,
  NothingFoundIllustration,
  RadarIllustration,
} from '@percona/peak-ui';
import type { FC } from 'react';
import { Messages } from '../hub.messages';
import type { HubMessages, HubTreeColumn, HubViewModel } from '../hub.types';
import { ClustersTable } from './ClustersTable';
import { HubActions } from './HubActions';
import { HubSection, SectionState } from './HubSection';

export interface ClustersSectionProps {
  view: HubViewModel;
  treeColumns: HubTreeColumn[];
  messages: HubMessages;
  onRetry: () => void;
  /** Clears the environment filter. */
  onShowAllEnvironments: () => void;
  /** Shows only the clusters behind on patches. */
  onShowPatches: () => void;
  /** Clears the behind on patches filter. */
  onClearPatches: () => void;
}

/** Clusters: the full-width tree, or how to start when nothing is monitored. */
export const ClustersSection: FC<ClustersSectionProps> = ({
  view,
  treeColumns,
  messages,
  onRetry,
  onShowAllEnvironments,
  onShowPatches,
  onClearPatches,
}) => {
  const { clusters, patchesFilter } = view;
  const behind = view.status.patches?.behind ?? 0;
  const clearPatches = () => {
    onClearPatches();
    // Without clusters behind the chip leaves with its filter, so focus moves to the section's heading instead of falling to the page.
    if (!behind) {
      document.getElementById('clusters-title')?.focus();
    }
  };
  const empty = view.status.state === 'empty';
  const { matchesNothing } = view.environment;
  const hasActions = view.actions.add.length > 0;

  return (
    <HubSection
      id="clusters"
      title={Messages.sections.clusters}
      count={empty || matchesNothing ? undefined : clusters.length}
      countLabel={Messages.sectionCount.clusters(
        clusters.length,
        messages.unit.cluster
      )}
      door={empty || matchesNothing ? undefined : view.sectionDoors.clusters}
    >
      <SectionState state={view.sections.clusters} onRetry={onRetry}>
        {empty ? (
          <Paper variant="outlined">
            <EmptyState
              size="medium"
              titleComponent="h3"
              illustration={<RadarIllustration fontSize="large" />}
              title={messages.empty.title}
              description={
                view.role === 'admin'
                  ? messages.empty.description
                  : messages.empty.viewer
              }
              actions={
                hasActions ? <HubActions actions={view.actions} /> : undefined
              }
            />
          </Paper>
        ) : matchesNothing ? (
          <Paper variant="outlined">
            <EmptyState
              size="small"
              titleComponent="h3"
              illustration={<NothingFoundIllustration />}
              title={Messages.environment.noMatch(messages.unit.cluster[1])}
              description={`${Messages.environment.noService(messages.name)} ${Messages.environment.pickAnother}`}
              actions={
                <Button
                  variant="outlined"
                  size="small"
                  onClick={onShowAllEnvironments}
                >
                  {Messages.environment.showAll}
                </Button>
              }
            />
          </Paper>
        ) : (
          <>
            {(patchesFilter || behind > 0) && (
              <Box sx={{ mb: 1.5 }}>
                {/* A toggle: the same chip filters and clears, so focus stays on it. */}
                <Chip
                  size="small"
                  label={Messages.clusters.patchesFilter.label(behind)}
                  aria-pressed={patchesFilter}
                  onClick={patchesFilter ? clearPatches : onShowPatches}
                  {...(patchesFilter ? { onDelete: clearPatches } : {})}
                />
              </Box>
            )}
            {patchesFilter && !clusters.length ? (
              <Paper variant="outlined">
                <EmptyState
                  size="small"
                  titleComponent="h3"
                  illustration={<NothingFoundIllustration />}
                  title={Messages.clusters.patchesFilter.none}
                  description={Messages.clusters.patchesFilter.noneDescription}
                  actions={
                    <Button
                      variant="outlined"
                      size="small"
                      onClick={clearPatches}
                    >
                      {Messages.clusters.patchesFilter.showAll}
                    </Button>
                  }
                />
              </Paper>
            ) : (
              <ClustersTable
                key={`${view.scenario}:${view.environment.selected ?? ''}:${patchesFilter}`}
                tableName={`hub-${view.technology.id}-clusters`}
                clusters={clusters}
                treeColumns={treeColumns}
                expanded={view.expanded}
                label={messages.clusters.tableLabel}
              />
            )}
          </>
        )}
      </SectionState>
    </HubSection>
  );
};

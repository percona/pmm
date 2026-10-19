import Button from '@mui/material/Button';
import Paper from '@mui/material/Paper';
import {
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
}

/** Clusters: the full-width tree, or how to start when nothing is monitored. */
export const ClustersSection: FC<ClustersSectionProps> = ({
  view,
  treeColumns,
  messages,
  onRetry,
  onShowAllEnvironments,
}) => {
  const { clusters } = view;
  const empty = view.status.state === 'empty';
  const { matchesNothing } = view.environment;
  const hasActions = !!(view.actions.addService || view.actions.install);

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
          <ClustersTable
            key={`${view.scenario}:${view.environment.selected ?? ''}`}
            tableName={`hub-${view.technology.id}-clusters`}
            clusters={clusters}
            treeColumns={treeColumns}
            expanded={view.expanded}
            label={messages.clusters.tableLabel}
          />
        )}
      </SectionState>
    </HubSection>
  );
};

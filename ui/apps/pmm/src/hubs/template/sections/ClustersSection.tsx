import Paper from '@mui/material/Paper';
import { EmptyState, RadarIllustration } from '@percona/peak-ui';
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
}

/** Clusters: the full-width tree, or how to start when nothing is monitored. */
export const ClustersSection: FC<ClustersSectionProps> = ({
  view,
  treeColumns,
  messages,
  onRetry,
}) => {
  const { clusters } = view;
  const empty = view.status.state === 'empty' || clusters.length === 0;
  const hasActions = !!(view.actions.addService || view.actions.install);

  return (
    <HubSection
      id="clusters"
      title={Messages.sections.clusters}
      count={empty ? undefined : clusters.length}
      countLabel={Messages.sectionCount.clusters(
        clusters.length,
        messages.unit.cluster
      )}
      door={empty ? undefined : view.sectionDoors.clusters}
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

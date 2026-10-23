import Box from '@mui/material/Box';
import { DEFAULT_PAGE_MAX_WIDTH } from '@percona/peak-ui';
import { useEffect } from 'react';
import { useLocation } from 'react-router-dom';
import { Page } from 'components/page';
import type { TechnologyHubConfig } from './hub.types';
import { AppsSection } from './sections/AppsSection';
import { BackupsSection } from './sections/BackupsSection';
import { ClustersSection } from './sections/ClustersSection';
import { DashboardsSection } from './sections/DashboardsSection';
import { DemoControls } from './sections/DemoControls';
import { HubBand } from './sections/HubBand';
import { HubHeader } from './sections/HubHeader';
import { HubStatusBanner } from './sections/HubStatusBanner';
import { NeedsAttentionSection } from './sections/NeedsAttentionSection';
import { QueriesSection } from './sections/QueriesSection';
import { HUB_CONTAINER } from './sections/hub.sx';
import { useTechnologyHub } from './useTechnologyHub';

// Two columns once each half holds the 500 px backups table beside the 32 px gap.
const TWO_COLUMNS_FROM = 1040;

// Each band takes its width from the Figma hub; the status banner sits at Peak's page width.
const STATUS_WIDTH = DEFAULT_PAGE_MAX_WIDTH;

// Prototype only: the dev server, or a build that asks for it (the demo host's), never a production one.
const SHOW_DEMO_CONTROLS =
  import.meta.env.DEV || import.meta.env.VITE_PMM_PROTOTYPE === 'true';

// A pasted or cut link can hold a lone "%", which decodeURIComponent throws on.
const sectionId = (hash: string) => {
  try {
    return decodeURIComponent(hash.slice(1));
  } catch {
    return hash.slice(1);
  }
};

// A door to `#clusters` and the like lands on that section and moves focus to its heading.
const useSectionFocus = () => {
  const { hash, key } = useLocation();

  useEffect(() => {
    if (!hash) {
      return;
    }

    const section = document.getElementById(sectionId(hash));
    const heading = section?.querySelector<HTMLElement>('h2');

    if (!section || !heading) {
      return;
    }

    section.scrollIntoView?.({ block: 'start' });
    heading.focus({ preventScroll: true });
  }, [hash, key]);
};

export interface HubPageProps<TSnapshot> {
  config: TechnologyHubConfig<TSnapshot>;
}

/** A technology hub: status, what needs attention, the estate, and the ways into it. */
export const HubPage = <TSnapshot,>({ config }: HubPageProps<TSnapshot>) => {
  const { view, params, setParams, refetch, freshness, isInitialLoading } =
    useTechnologyHub(config);
  const { messages } = config;

  useSectionFocus();

  return (
    <Page title={messages.name} hideTitle maxWidth="full">
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
          // Sections lay out by the hub's width, which the sidebar narrows, not by the viewport's.
          containerType: 'inline-size',
          containerName: HUB_CONTAINER,
        }}
      >
        <HubHeader
          view={view}
          icon={config.icon}
          messages={messages}
          setParams={setParams}
          freshness={freshness}
          refetch={refetch}
        />
        <HubBand maxWidth={STATUS_WIDTH}>
          <HubStatusBanner
            status={view.status}
            name={messages.name}
            unit={messages.unit.cluster}
            matchesNothing={view.environment.matchesNothing}
            loading={isInitialLoading}
            issues={view.attention.length}
          />
        </HubBand>
        {view.status.state !== 'empty' &&
          !(view.environment.matchesNothing && !view.attention.length) && (
            <HubBand>
              <NeedsAttentionSection
                items={view.attention}
                state={view.sections.attention}
                door={view.sectionDoors.attention}
                checkedAt={view.status.checkedAt}
                coverage={view.alertCoverage}
                onRetry={refetch}
              />
            </HubBand>
          )}
        <HubBand>
          <ClustersSection
            view={view}
            treeColumns={config.treeColumns}
            messages={messages}
            onRetry={refetch}
            onShowAllEnvironments={() => setParams({ environment: null })}
            onShowPatches={() => setParams({ patches: 'behind' })}
            onClearPatches={() => setParams({ patches: null })}
          />
        </HubBand>
        <HubBand>
          <Box
            sx={(theme) => ({
              display: 'grid',
              gridTemplateColumns: 'minmax(0, 1fr)',
              [theme.containerQueries(HUB_CONTAINER).up(TWO_COLUMNS_FROM)]: {
                gridTemplateColumns: 'repeat(2, minmax(0, 1fr))',
              },
              alignItems: 'start',
              gap: 4,
            })}
          >
            <BackupsSection
              technologyId={view.technology.id}
              backups={view.backups}
              state={view.sections.backups}
              messages={messages}
              matchesNothing={view.environment.matchesNothing}
              onRetry={refetch}
            />
            <AppsSection
              apps={view.apps}
              door={view.sectionDoors.apps}
              state={view.sections.apps}
              onRetry={refetch}
            />
            <QueriesSection
              queries={view.queries}
              state={view.sections.queries}
              empty={
                view.status.state === 'empty' || view.environment.matchesNothing
              }
              onRetry={refetch}
            />
            <DashboardsSection
              dashboards={view.dashboards}
              state={view.sections.dashboards}
              onRetry={refetch}
            />
          </Box>
        </HubBand>
        {SHOW_DEMO_CONTROLS && (
          <DemoControls
            name={messages.name}
            params={params}
            setParams={setParams}
            operations={view.capabilities.operations}
            admin={view.role === 'admin'}
          />
        )}
      </Box>
    </Page>
  );
};

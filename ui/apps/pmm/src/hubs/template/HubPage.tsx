import Box from '@mui/material/Box';
import { DEFAULT_PAGE_MAX_WIDTH } from '@percona/peak-ui';
import { useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { Page } from 'components/page';
import { attentionIssues } from './hub.attention';
import type { HubScenario, TechnologyHubConfig } from './hub.types';
import { AppsSection } from './sections/AppsSection';
import { DashboardsSection } from './sections/DashboardsSection';
import { DemoControls } from './sections/DemoControls';
import { HubHeader } from './sections/HubHeader';
import { HubRow } from './sections/HubRow';
import { HubStatusBanner } from './sections/HubStatusBanner';
import { NeedsAttentionSection } from './sections/NeedsAttentionSection';
import { QueriesSection } from './sections/QueriesSection';
import { HUB_CONTAINER } from './sections/hub.sx';
import { useTechnologyHub } from './useTechnologyHub';

// The whole hub, header included, so the title and every section share one left edge.
const HUB_MAX_WIDTH = 1440;

// The status banner sits centred at Peak's page width.
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

/** A technology hub: status, what needs attention, and the ways into the technology's data and apps. */
export const HubPage = <TSnapshot,>({ config }: HubPageProps<TSnapshot>) => {
  const { view, params, setParams, refetch, freshness, isInitialLoading } =
    useTechnologyHub(config);
  const { messages } = config;
  // How far Needs attention is open; it folds back to its first rows when the scenario changes.
  const [attentionOpen, setAttentionOpen] = useState<{
    scenario: HubScenario;
    revealed: number;
  } | null>(null);
  const attentionRevealed =
    attentionOpen?.scenario === view.scenario ? attentionOpen.revealed : 0;
  const revealAttention = (revealed: number) =>
    setAttentionOpen({ scenario: view.scenario, revealed });
  // With nothing wrong, the section is not there, nor its row; an empty estate has nothing to check.
  const showsAttention =
    view.status.state !== 'empty' &&
    (view.attention.length > 0 || view.sections.attention.status !== 'ready');

  useSectionFocus();

  return (
    <Page title={messages.name} hideTitle maxWidth="full">
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
          width: '100%',
          maxWidth: HUB_MAX_WIDTH,
          mx: 'auto',
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
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 4, py: 2 }}>
          <Box sx={{ width: '100%', maxWidth: STATUS_WIDTH, mx: 'auto' }}>
            <HubStatusBanner
              status={view.status}
              name={messages.name}
              unit={messages.unit.cluster}
              matchesNothing={view.environment.matchesNothing}
              loading={isInitialLoading}
              issues={attentionIssues(view.attention)}
              // Review N issues opens every row before it scrolls to them.
              onReview={() => revealAttention(Number.POSITIVE_INFINITY)}
            />
          </Box>
          {/* Recommendations will take the right side; until then the row is only there for Needs attention. */}
          {showsAttention && (
            <HubRow
              left={
                <NeedsAttentionSection
                  items={view.attention}
                  state={view.sections.attention}
                  environmentFiltered={view.environment.selected !== null}
                  describeDestination={config.describeDestination}
                  revealed={attentionRevealed}
                  onRevealedChange={revealAttention}
                  onRetry={refetch}
                />
              }
            />
          )}
          <HubRow
            left={
              <QueriesSection
                queries={view.queries}
                state={view.sections.queries}
                empty={
                  view.status.state === 'empty' ||
                  view.environment.matchesNothing
                }
                onRetry={refetch}
              />
            }
            right={
              <>
                <AppsSection
                  apps={view.apps}
                  door={view.sectionDoors.apps}
                  state={view.sections.apps}
                  onRetry={refetch}
                />
                <DashboardsSection
                  dashboards={view.dashboards}
                  state={view.sections.dashboards}
                  onRetry={refetch}
                />
              </>
            }
          />
        </Box>
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

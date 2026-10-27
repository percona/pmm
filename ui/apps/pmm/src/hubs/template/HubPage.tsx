import Box from '@mui/material/Box';
import { DEFAULT_PAGE_MAX_WIDTH } from '@percona/peak-ui';
import { useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { Footer } from 'components/footer';
import { Page } from 'components/page';
import { attentionIssues } from './hub.attention';
import type { HubScenario, TechnologyHubConfig } from './hub.types';
import { DashboardsSection } from './sections/DashboardsSection';
import { DemoControls } from './sections/DemoControls';
import { HubHeader } from './sections/HubHeader';
import { HubRow } from './sections/HubRow';
import { HubShortcuts } from './sections/HubShortcuts';
import { HubStatusBanner } from './sections/HubStatusBanner';
import { HubWatermark } from './sections/HubWatermark';
import { NeedsAttentionSection } from './sections/NeedsAttentionSection';
import { QueriesSection } from './sections/QueriesSection';
import { RecommendationsSection } from './sections/RecommendationsSection';
import { HUB_CONTAINER } from './sections/hub.sx';
import { useTechnologyHub } from './useTechnologyHub';

// The whole hub, header included, so the title and every section share one left edge.
const HUB_MAX_WIDTH = 1440;

// The status banner sits centred at Peak's page width.
const STATUS_WIDTH = DEFAULT_PAGE_MAX_WIDTH;

// Recommendations shows as many rows as Needs attention beside it, and three lines of two alone.
const RECOMMENDATIONS_VISIBLE = { beside: 5, alone: 6 };

// Prototype only: the dev server, or a build that asks for it (the demo host's), never a production one.
const PROTOTYPE_BUILD =
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

/** A technology hub: shortcuts, status, what needs attention, and the ways into the technology's data. */
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
  // With nothing wrong, the section is not there; an empty estate has nothing to check.
  const showsAttention =
    view.status.state !== 'empty' &&
    (view.attention.length > 0 || view.sections.attention.status !== 'ready');
  const showsRecommendations =
    view.status.state !== 'empty' &&
    (view.recommendations.length > 0 ||
      view.sections.recommendations.status !== 'ready');
  const recommendationsVisible = showsAttention
    ? RECOMMENDATIONS_VISIBLE.beside
    : RECOMMENDATIONS_VISIBLE.alone;
  // Recommendations folds back to its first rows when the scenario changes.
  const [recommendationsOpenFor, setRecommendationsOpenFor] =
    useState<HubScenario | null>(null);
  const recommendationsExpanded = recommendationsOpenFor === view.scenario;
  // Folded, the two lists' boxes match in height so their footers line up; an open one grows alone.
  const matchLists = attentionRevealed === 0 && !recommendationsExpanded;

  useSectionFocus();

  return (
    // The footer goes inside the hub's container, aligned with everything else.
    <Page title={messages.name} hideTitle maxWidth="full" footer={null}>
      <Box
        sx={{
          position: 'relative',
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
        {/* First, so a screen reader says it before the page's title. */}
        {PROTOTYPE_BUILD && <HubWatermark />}
        <HubHeader
          view={view}
          icon={config.icon}
          messages={messages}
          setParams={setParams}
          freshness={freshness}
          refetch={refetch}
        />
        <HubShortcuts
          shortcuts={view.shortcuts}
          describeDestination={config.describeDestination}
        />
        {/* A little more room above the banner than below it sets it off from the tiles. */}
        <Box
          sx={{
            display: 'flex',
            flexDirection: 'column',
            gap: 4,
            pt: 3,
            pb: 2,
          }}
        >
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
          {(showsAttention || showsRecommendations) && (
            <HubRow
              stretch={matchLists}
              left={
                showsAttention && (
                  <NeedsAttentionSection
                    items={view.attention}
                    state={view.sections.attention}
                    environmentFiltered={view.environment.selected !== null}
                    describeDestination={config.describeDestination}
                    revealed={attentionRevealed}
                    onRevealedChange={revealAttention}
                    onRetry={refetch}
                  />
                )
              }
              right={
                showsRecommendations && (
                  <RecommendationsSection
                    items={view.recommendations}
                    state={view.sections.recommendations}
                    environmentFiltered={view.environment.selected !== null}
                    describeDestination={config.describeDestination}
                    visibleCount={recommendationsVisible}
                    wide={!showsAttention}
                    expanded={recommendationsExpanded}
                    onExpandedChange={(expanded) =>
                      setRecommendationsOpenFor(expanded ? view.scenario : null)
                    }
                    onRetry={refetch}
                  />
                )
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
              <DashboardsSection
                dashboards={view.dashboards}
                allApps={view.sectionDoors.apps}
                state={view.sections.dashboards}
                onRetry={refetch}
              />
            }
          />
        </Box>
        <Footer />
        {PROTOTYPE_BUILD && (
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

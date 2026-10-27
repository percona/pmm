import Box from '@mui/material/Box';
import Paper from '@mui/material/Paper';
import { type FC, useEffect, useId, useRef } from 'react';
import { Messages } from '../hub.messages';
import type {
  HubDestination,
  HubDestinationLook,
  HubRecommendation,
  HubRecommendationKind,
  HubSectionState,
} from '../hub.types';
import { HubListRow, ShowMoreButton } from './HubListRow';
import { HubSection, SectionState } from './HubSection';
import { HUB_TWO_COLUMNS_FROM } from './HubRow';
import { HUB_CONTAINER } from './hub.sx';

export const RECOMMENDATIONS_SECTION_ID = 'recommendations';

const SEVERITY = {
  notset: 'notSetUp',
  info: 'info',
} as const satisfies Record<HubRecommendationKind, string>;

export interface RecommendationsSectionProps {
  items: HubRecommendation[];
  state: HubSectionState;
  /** One environment is selected, so the rows don't repeat it. */
  environmentFiltered: boolean;
  describeDestination: (destination: HubDestination) => HubDestinationLook;
  /** Rows shown before "Show N more": fewer while Needs attention has more to say. */
  visibleCount: number;
  /** Alone in its row, it takes both columns and puts two rows on a line. */
  wide: boolean;
  expanded: boolean;
  onExpandedChange: (expanded: boolean) => void;
  onRetry: () => void;
}

/** Recommendations: what is fine today but worth improving, one row per kind; with nothing to suggest, the section is not there at all. */
export const RecommendationsSection: FC<RecommendationsSectionProps> = ({
  items,
  state,
  environmentFiltered,
  describeDestination,
  visibleCount,
  wide,
  expanded,
  onExpandedChange,
  onRetry,
}) => {
  const listId = useId();
  const hidden = Math.max(0, items.length - visibleCount);
  const firstRevealed = useRef<HTMLLIElement>(null);
  const revealing = useRef(false);

  useEffect(() => {
    if (expanded && revealing.current) {
      revealing.current = false;
      firstRevealed.current?.focus();
    }
  }, [expanded]);

  if (state.status === 'ready' && !items.length) {
    return null;
  }

  const visible = expanded ? items : items.slice(0, visibleCount);

  return (
    <HubSection
      id={RECOMMENDATIONS_SECTION_ID}
      title={Messages.sections.recommendations}
    >
      <SectionState state={state} onRetry={onRetry} skeletonRows={2}>
        {items.length > 0 && (
          // Fills a stretched row, so its footer meets the bottom beside Needs attention's.
          <Paper
            variant="outlined"
            sx={{
              overflow: 'hidden',
              flex: 1,
              display: 'flex',
              flexDirection: 'column',
            }}
          >
            <Box
              component="ul"
              id={listId}
              sx={(theme) => ({
                m: 0,
                p: 0,
                listStyle: 'none',
                display: 'grid',
                gridTemplateColumns: 'minmax(0, 1fr)',
                // The rules between rows are the list's own colour showing through 1px gaps, so they hold on one line or two.
                gap: '1px',
                bgcolor: 'divider',
                ...(wide && {
                  [theme
                    .containerQueries(HUB_CONTAINER)
                    .up(HUB_TWO_COLUMNS_FROM)]: {
                    gridTemplateColumns: 'repeat(2, minmax(0, 1fr))',
                    '& > li:last-of-type:nth-of-type(odd)': {
                      gridColumn: '1 / -1',
                    },
                  },
                }),
              })}
            >
              {visible.map((item, index) => (
                <HubListRow
                  key={`${item.kind}:${item.title}`}
                  testId="recommendation-row"
                  severity={SEVERITY[item.kind]}
                  severityLabel={Messages.recommendations.kind[item.kind]}
                  title={item.title}
                  quiet
                  impact={item.impact}
                  meta={[
                    item.location,
                    !environmentFiltered && item.environment,
                  ]}
                  destination={item.destination}
                  look={describeDestination(item.destination)}
                  rowRef={
                    expanded && index === visibleCount
                      ? firstRevealed
                      : undefined
                  }
                />
              ))}
            </Box>
            {hidden > 0 && (
              <ShowMoreButton
                label={
                  expanded
                    ? Messages.attention.showFewer
                    : Messages.attention.showMore(hidden)
                }
                expanded={expanded}
                controls={listId}
                onClick={() => {
                  revealing.current = !expanded;
                  onExpandedChange(!expanded);
                }}
              />
            )}
          </Paper>
        )}
      </SectionState>
    </HubSection>
  );
};

import NotificationsOutlined from '@mui/icons-material/NotificationsOutlined';
import Box from '@mui/material/Box';
import Link from '@mui/material/Link';
import Paper from '@mui/material/Paper';
import { type FC, useEffect, useId, useRef } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { attentionIssues } from '../hub.attention';
import { alertsLink, ATTENTION_SECTION_ID } from '../hub.doors';
import { Messages } from '../hub.messages';
import type {
  HubDestination,
  HubDestinationLook,
  HubNeedsAttentionItem,
  HubSectionState,
} from '../hub.types';
import { aboveRow, focusRing, HubListRow, ShowMoreButton } from './HubListRow';
import { HubSection, SectionState } from './HubSection';

/** Rows shown before "Show N more". */
export const ATTENTION_CAP = 5;
/** Rows each "Show N more" adds. */
const ATTENTION_STEP = 5;
/** A rest this small shows at once, so no click reveals just one or two rows. */
const ATTENTION_LAST_STEP = 7;

/** How many rows each "Show N more" reveals, in order: steps of 5, the last one up to 7. */
export const attentionSteps = (count: number): number[] => {
  const steps: number[] = [];

  for (
    let rest = count - ATTENTION_CAP;
    rest > 0;
    rest -= steps[steps.length - 1]
  ) {
    steps.push(rest <= ATTENTION_LAST_STEP ? rest : ATTENTION_STEP);
  }

  return steps;
};

const A = Messages.attention;

export interface NeedsAttentionSectionProps {
  items: HubNeedsAttentionItem[];
  state: HubSectionState;
  /** One environment is selected, so the rows don't repeat it. */
  environmentFiltered: boolean;
  /** The sidebar's icon for a destination and its place there. */
  describeDestination: (destination: HubDestination) => HubDestinationLook;
  /** How many "Show N more" steps are open; Infinity opens them all. */
  revealed: number;
  onRevealedChange: (revealed: number) => void;
  onRetry: () => void;
}

/** The muted line under the impact: where, which environment, how long, and the firing alert. */
const metaOf = (item: HubNeedsAttentionItem, environment: boolean) => [
  // The title usually names the cluster; a path or a list adds to it.
  item.location && !item.title.includes(item.location) && item.location,
  environment && item.environment,
  item.since && A.since(item.since),
  item.alertFiring && (
    <Link
      key="alert"
      component={RouterLink}
      to={alertsLink(item.alertService)}
      color="inherit"
      underline="hover"
      sx={(theme) => ({
        ...aboveRow,
        display: 'inline-flex',
        alignItems: 'center',
        gap: 0.25,
        verticalAlign: 'top',
        borderRadius: `${theme.shape.borderRadiusXs}px`,
        '&:focus-visible': { ...focusRing(theme), outlineOffset: 1 },
      })}
    >
      <NotificationsOutlined sx={{ fontSize: 16 }} />
      {A.alertFiring}
    </Link>
  ),
];

/** Needs attention: what is wrong right now, worst first; with nothing wrong, the section is not there at all. */
export const NeedsAttentionSection: FC<NeedsAttentionSectionProps> = ({
  items,
  state,
  environmentFiltered,
  describeDestination,
  revealed,
  onRevealedChange,
  onRetry,
}) => {
  const listId = useId();
  const steps = attentionSteps(items.length);
  const open = Math.min(revealed, steps.length);
  const shown =
    ATTENTION_CAP + steps.slice(0, open).reduce((sum, step) => sum + step, 0);
  const allShown = open === steps.length;
  const issues = attentionIssues(items);
  // The revealed rows sit above the button, so focus moves to the first of them instead of past them.
  const firstRevealed = useRef<HTMLLIElement>(null);
  const revealing = useRef(false);

  useEffect(() => {
    if (revealing.current) {
      revealing.current = false;
      firstRevealed.current?.focus();
    }
  }, [open]);

  if (state.status === 'ready' && !items.length) {
    return null;
  }

  const visible = items.slice(0, shown);
  const firstNew = open > 0 ? shown - steps[open - 1] : -1;

  return (
    <HubSection
      id={ATTENTION_SECTION_ID}
      title={Messages.sections.attention}
      count={items.length ? issues : undefined}
      countLabel={Messages.sectionCount.attention(issues)}
      showCountLabel
    >
      <SectionState state={state} onRetry={onRetry} skeletonRows={2}>
        {items.length > 0 && (
          // Fills a stretched row, so its footer meets the bottom beside Recommendations'.
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
              sx={{ m: 0, p: 0, listStyle: 'none' }}
            >
              {visible.map((item, index) => (
                <HubListRow
                  key={`${item.severity}:${item.title}`}
                  testId="attention-row"
                  severity={item.severity}
                  severityLabel={A.severity[item.severity]}
                  title={item.title}
                  impact={item.impact}
                  meta={metaOf(item, !environmentFiltered)}
                  destination={item.destination}
                  look={describeDestination(item.destination)}
                  divider={index < visible.length - 1}
                  rowRef={index === firstNew ? firstRevealed : undefined}
                />
              ))}
            </Box>
            {steps.length > 0 && (
              <ShowMoreButton
                label={allShown ? A.showFewer : A.showMore(steps[open])}
                expanded={allShown}
                controls={listId}
                onClick={() => {
                  revealing.current = !allShown;
                  onRevealedChange(allShown ? 0 : open + 1);
                }}
              />
            )}
          </Paper>
        )}
      </SectionState>
    </HubSection>
  );
};

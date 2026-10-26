import ArrowDownward from '@mui/icons-material/ArrowDownward';
import ArrowForward from '@mui/icons-material/ArrowForward';
import ArrowOutward from '@mui/icons-material/ArrowOutward';
import NotificationsOutlined from '@mui/icons-material/NotificationsOutlined';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Link from '@mui/material/Link';
import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';
import type { Theme } from '@mui/material/styles';
import { SeverityIcon, Tooltip } from '@percona/peak-ui';
import { ThemeScope } from 'components/theme-scope';
import {
  type FC,
  Fragment,
  type ReactNode,
  type Ref,
  useEffect,
  useId,
  useRef,
  useState,
} from 'react';
import { Link as RouterLink, useLocation } from 'react-router-dom';
import { attentionIssues } from '../hub.attention';
import { alertsLink, ATTENTION_SECTION_ID, withQuery } from '../hub.doors';
import { Messages } from '../hub.messages';
import type {
  HubDestination,
  HubDestinationLook,
  HubNeedsAttentionItem,
  HubSectionState,
} from '../hub.types';
import { HubSection, SectionState } from './HubSection';
import { HUB_CONTAINER, visuallyHidden } from './hub.sx';

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

const ARROWS: Record<HubDestination['kind'], ReactNode> = {
  page: <ArrowForward />,
  inpage: <ArrowDownward />,
  external: <ArrowOutward />,
};

// Above the action's click area, which covers the whole row.
const aboveRow = { position: 'relative', zIndex: 1 } as const;

const focusRing = (theme: Theme) => ({
  outline: `2px solid ${theme.palette.primary.main}`,
  outlineOffset: -2,
});

/** Where the action goes: a section of this page keeps the hub's params, another page takes the destination's. */
const useHref = ({ route, params, kind }: HubDestination) => {
  const { search } = useLocation();

  return kind === 'inpage' ? `${search}${route}` : withQuery(route, params);
};

/**
 * The row's one action, a small text button: the sidebar's icon, the destination and an arrow for where it goes.
 * Its link covers the row, so the button's face sits above that and keeps a hover of its own.
 */
const RowAction: FC<{
  destination: HubDestination;
  look: HubDestinationLook;
  linkRef: Ref<HTMLAnchorElement>;
}> = ({ destination, look, linkRef }) => {
  const href = useHref(destination);
  const [hovered, setHovered] = useState(false);
  const [focused, setFocused] = useState(false);
  const external = destination.kind === 'external';
  const Icon = look.icon;

  return (
    <Button
      ref={linkRef}
      // React Router renders an absolute URL as a plain link out.
      component={RouterLink}
      to={href}
      {...(external ? { target: '_blank', rel: 'noopener noreferrer' } : {})}
      size="small"
      variant="text"
      disableRipple
      onFocus={(event) =>
        setFocused(event.currentTarget.matches(':focus-visible'))
      }
      onBlur={() => setFocused(false)}
      sx={(theme) => ({
        // Static, so the click area below spans the row rather than the button.
        position: 'static',
        flexShrink: 0,
        p: 0,
        minWidth: 0,
        '&:hover, &.Mui-focusVisible': {
          backgroundColor: 'transparent',
          color: 'primary.main',
        },
        '&::after': { content: '""', position: 'absolute', inset: 0 },
        '&.Mui-focusVisible::after': focusRing(theme),
      })}
    >
      {/* The tooltip listens to the visible button alone, not to the whole row the link covers. */}
      <Tooltip
        title={look.path.join(' › ')}
        open={hovered || focused}
        disableHoverListener
        disableFocusListener
        disableTouchListener
        describeChild
      >
        <Box
          component="span"
          onMouseEnter={() => setHovered(true)}
          onMouseLeave={() => setHovered(false)}
          sx={(theme) => ({
            ...aboveRow,
            display: 'inline-flex',
            alignItems: 'center',
            gap: 0.5,
            minHeight: 24,
            px: 0.5,
            borderRadius: `${theme.shape.borderRadiusFull}px`,
            '& .MuiSvgIcon-root': { fontSize: 16 },
            '&:hover': {
              backgroundColor: theme.palette.primary.hover,
              color: theme.palette.primary.light,
            },
            '.Mui-focusVisible > &': {
              backgroundColor: theme.palette.primary.focusVisible,
            },
          })}
        >
          {Icon && <Icon />}
          {destination.label}
          {ARROWS[destination.kind]}
          {external && (
            <Box component="span" sx={visuallyHidden}>
              {` ${Messages.newTab}`}
            </Box>
          )}
        </Box>
      </Tooltip>
    </Button>
  );
};

/** The muted line under the impact: where, which environment, how long, and the firing alert. */
const MetaLine: FC<{ item: HubNeedsAttentionItem; environment: boolean }> = ({
  item,
  environment,
}) => {
  const parts: ReactNode[] = [
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
  ].filter(Boolean);

  return parts.length ? (
    <Typography variant="caption" color="text.secondary" component="p">
      {parts.map((part, index) => (
        <Fragment key={index}>
          {index > 0 && ' · '}
          {part}
        </Fragment>
      ))}
    </Typography>
  ) : null;
};

const AttentionRow: FC<{
  item: HubNeedsAttentionItem;
  look: HubDestinationLook;
  environment: boolean;
  divider: boolean;
  /** Set on the first row a "Show N more" reveals, which then takes focus. */
  rowRef?: Ref<HTMLLIElement>;
}> = ({ item, look, environment, divider, rowRef }) => {
  const link = useRef<HTMLAnchorElement>(null);
  const severity = A.severity[item.severity];

  return (
    <Box
      component="li"
      ref={rowRef}
      tabIndex={rowRef ? -1 : undefined}
      data-testid="attention-row"
      data-severity={item.severity}
      sx={(theme) => ({
        position: 'relative',
        display: 'flex',
        flexDirection: 'column',
        gap: 0.25,
        px: 1.5,
        py: 1.5,
        ...(divider && { borderBottom: `1px solid ${theme.palette.divider}` }),
        '&:hover': { bgcolor: 'action.hover' },
        '&:focus-visible': focusRing(theme),
      })}
    >
      {/* The severity and the title head the row; what follows uses its full width. */}
      <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 1 }}>
        <Tooltip title={severity}>
          {/* Above the action's click area for its tooltip, so it passes clicks on to the action. */}
          <Box
            onClick={() => link.current?.click()}
            sx={{ ...aboveRow, display: 'flex', my: '-1px', cursor: 'pointer' }}
          >
            <SeverityIcon severity={item.severity} />
          </Box>
        </Tooltip>
        <Typography variant="body1" sx={{ fontWeight: 'fontWeightMedium' }}>
          <Box component="span" sx={visuallyHidden}>
            {`${severity}: `}
          </Box>
          {item.title}
        </Typography>
      </Box>
      <Box
        sx={(theme) => ({
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'flex-start',
          gap: 1,
          [theme.containerQueries(HUB_CONTAINER).up('sm')]: {
            flexDirection: 'row',
            alignItems: 'center',
            gap: 3,
          },
        })}
      >
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Typography variant="body2" color="text.secondary">
            {item.impact}
          </Typography>
          <MetaLine item={item} environment={environment} />
        </Box>
        <RowAction destination={item.destination} look={look} linkRef={link} />
      </Box>
    </Box>
  );
};

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
          <Paper variant="outlined" sx={{ overflow: 'hidden' }}>
            <Box
              component="ul"
              id={listId}
              sx={{ m: 0, p: 0, listStyle: 'none' }}
            >
              {visible.map((item, index) => (
                <AttentionRow
                  key={`${item.severity}:${item.title}`}
                  item={item}
                  look={describeDestination(item.destination)}
                  environment={!environmentFiltered}
                  divider={index < visible.length - 1 || steps.length > 0}
                  rowRef={index === firstNew ? firstRevealed : undefined}
                />
              ))}
            </Box>
            {steps.length > 0 && (
              <Box sx={{ px: 1, py: 0.5 }}>
                {/* Neutral, Peak's Base theme: a quiet control under the rows. */}
                <ThemeScope theme="base">
                  <Button
                    size="small"
                    variant="text"
                    aria-expanded={allShown}
                    aria-controls={listId}
                    onClick={() => {
                      revealing.current = !allShown;
                      onRevealedChange(allShown ? 0 : open + 1);
                    }}
                  >
                    {allShown ? A.showFewer : A.showMore(steps[open])}
                  </Button>
                </ThemeScope>
              </Box>
            )}
          </Paper>
        )}
      </SectionState>
    </HubSection>
  );
};

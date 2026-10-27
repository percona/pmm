import ArrowDownward from '@mui/icons-material/ArrowDownward';
import ArrowForward from '@mui/icons-material/ArrowForward';
import ArrowOutward from '@mui/icons-material/ArrowOutward';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Typography, { type TypographyProps } from '@mui/material/Typography';
import type { Theme } from '@mui/material/styles';
import {
  SeverityIcon,
  type SeverityIconSeverity,
  severityTextColor,
  Tooltip,
} from '@percona/peak-ui';
import { ThemeScope } from 'components/theme-scope';
import {
  type FC,
  Fragment,
  type ReactNode,
  type Ref,
  useRef,
  useState,
} from 'react';
import { Link as RouterLink, useLocation } from 'react-router-dom';
import { withQuery } from '../hub.doors';
import { Messages } from '../hub.messages';
import type { HubDestination, HubDestinationLook } from '../hub.types';
import { visuallyHidden } from './hub.sx';

// The rows of Needs attention and Recommendations: a severity and a title head the row, then the impact, a muted meta line and one action.

// A row lays out by its own width, as it sits in a wide column, a narrow one, or half of a wide one.
const ROW_CONTAINER = 'hub-row';
/** From this width inside the row's padding, the action sits beside the text rather than under it. */
const ACTION_BESIDE_FROM = 560;

const ARROWS: Record<HubDestination['kind'], ReactNode> = {
  page: <ArrowForward />,
  inpage: <ArrowDownward />,
  external: <ArrowOutward />,
};

/** Above the action's click area, which covers the whole row. */
export const aboveRow = { position: 'relative', zIndex: 1 } as const;

export const focusRing = (theme: Theme) => ({
  outline: `2px solid ${theme.palette.primary.main}`,
  outlineOffset: -2,
});

/** Where a destination goes: a section of this page keeps the hub's params, another page takes the destination's. */
export const useDestinationHref = ({ route, params, kind }: HubDestination) => {
  const { search } = useLocation();

  return kind === 'inpage' ? `${search}${route}` : withQuery(route, params);
};

/**
 * The row's one action, a small text button in Peak's neutral Base theme: the sidebar's icon, the destination and an arrow for where it goes.
 * Its link covers the row, so the button's face sits above that and keeps a hover of its own.
 */
const RowAction: FC<{
  destination: HubDestination;
  look: HubDestinationLook;
  linkRef: Ref<HTMLAnchorElement>;
}> = ({ destination, look, linkRef }) => {
  const href = useDestinationHref(destination);
  const [hovered, setHovered] = useState(false);
  const [focused, setFocused] = useState(false);
  const external = destination.kind === 'external';
  const Icon = look.icon;

  return (
    <ThemeScope theme="base">
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
          // Under the text, the face's own padding pulls back, so its icon lines up with the text above.
          ml: -0.5,
          [theme.containerQueries(ROW_CONTAINER).up(ACTION_BESIDE_FROM)]: {
            ml: 0,
          },
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
    </ThemeScope>
  );
};

/** One line, cut with an ellipsis; the full text shows in a tooltip while it is cut. Above the row's click area for the tooltip, so it passes clicks on. */
const OneLine: FC<
  TypographyProps & { text: string; onActivate: () => void }
> = ({ text, onActivate, sx, children, ...rest }) => {
  const [open, setOpen] = useState(false);

  return (
    <Tooltip
      title={text}
      open={open}
      disableHoverListener
      disableFocusListener
      disableTouchListener
      describeChild
    >
      <Typography
        noWrap
        onMouseEnter={(event) =>
          setOpen(
            event.currentTarget.scrollWidth > event.currentTarget.clientWidth
          )
        }
        onMouseLeave={() => setOpen(false)}
        onClick={onActivate}
        sx={[
          { ...aboveRow, cursor: 'pointer' },
          ...(Array.isArray(sx) ? sx : [sx]),
        ]}
        {...rest}
      >
        {children ?? text}
      </Typography>
    </Tooltip>
  );
};

export interface HubListRowProps {
  severity: SeverityIconSeverity;
  /** The severity in words, for its tooltip and for assistive technology. */
  severityLabel: string;
  title: string;
  /** A regular-weight title, for a list quieter than Needs attention. */
  quiet?: boolean;
  impact: string;
  /** The muted line's parts, joined with " · ". */
  meta: ReactNode[];
  destination: HubDestination;
  look: HubDestinationLook;
  /** A rule under the row; a list that draws its own lines leaves it out. */
  divider?: boolean;
  /** Set on a row that takes focus as Show more reveals it. */
  rowRef?: Ref<HTMLLIElement>;
  testId?: string;
}

/** One row: the severity and title head it, the impact and meta line follow at full width, and the action's click area covers it all. Every line is one line, so rows share one height. */
export const HubListRow: FC<HubListRowProps> = ({
  severity,
  severityLabel,
  title,
  quiet = false,
  impact,
  meta,
  destination,
  look,
  divider = false,
  rowRef,
  testId,
}) => {
  const link = useRef<HTMLAnchorElement>(null);
  const parts = meta.filter(Boolean);
  const activate = () => link.current?.click();

  return (
    <Box
      component="li"
      ref={rowRef}
      tabIndex={rowRef ? -1 : undefined}
      data-testid={testId}
      data-severity={severity}
      sx={(theme) => ({
        position: 'relative',
        display: 'flex',
        flexDirection: 'column',
        gap: 0.25,
        containerType: 'inline-size',
        containerName: ROW_CONTAINER,
        px: 1.5,
        py: 1.5,
        // Its own surface, so a list that draws its lines in the gaps between rows shows none through it.
        bgcolor: 'background.paper',
        ...(divider && { borderBottom: `1px solid ${theme.palette.divider}` }),
        '&:hover': {
          backgroundImage: `linear-gradient(${theme.palette.action.hover}, ${theme.palette.action.hover})`,
        },
        '&:focus-visible': focusRing(theme),
      })}
    >
      <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 1 }}>
        <Tooltip title={severityLabel}>
          {/* Above the action's click area for its tooltip, so it passes clicks on to the action. */}
          <Box
            onClick={activate}
            sx={{
              ...aboveRow,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              // One 24px column whatever the glyph's size, so titles line up across rows.
              width: 24,
              height: 24,
              my: '-1px',
              flexShrink: 0,
              cursor: 'pointer',
            }}
          >
            <SeverityIcon severity={severity} />
          </Box>
        </Tooltip>
        <OneLine
          text={title}
          onActivate={activate}
          variant="body1"
          sx={(theme) => ({
            flex: 1,
            minWidth: 0,
            fontWeight: quiet ? 'fontWeightRegular' : 'fontWeightMedium',
            // The icon's color, so the title reads as part of it.
            color: severityTextColor(severity, theme.palette.mode),
          })}
        >
          <Box component="span" sx={visuallyHidden}>
            {`${severityLabel}: `}
          </Box>
          {title}
        </OneLine>
      </Box>
      <Box
        sx={(theme) => ({
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'flex-start',
          gap: 1,
          [theme.containerQueries(ROW_CONTAINER).up(ACTION_BESIDE_FROM)]: {
            flexDirection: 'row',
            alignItems: 'center',
            gap: 3,
          },
        })}
      >
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <OneLine
            text={impact}
            onActivate={activate}
            variant="body2"
            color="text.secondary"
          />
          {/* Always a line, even an empty one, so every row keeps the same height. */}
          <Typography
            variant="caption"
            color="text.secondary"
            component="p"
            noWrap
            sx={{ minHeight: '1lh' }}
          >
            {parts.map((part, index) => (
              <Fragment key={index}>
                {index > 0 && ' · '}
                {part}
              </Fragment>
            ))}
          </Typography>
        </Box>
        <RowAction destination={destination} look={look} linkRef={link} />
      </Box>
    </Box>
  );
};

export interface ShowMoreButtonProps {
  label: string;
  /** Every row shows. */
  expanded: boolean;
  /** The list it opens. */
  controls: string;
  onClick: () => void;
}

/** The footer under a list's rows: neutral, Peak's Base theme, a quiet control. */
export const ShowMoreButton: FC<ShowMoreButtonProps> = ({
  label,
  expanded,
  controls,
  onClick,
}) => (
  // At the bottom of its box, under a rule, however tall the box is.
  <Box
    sx={{ mt: 'auto', px: 1, py: 0.5, borderTop: 1, borderColor: 'divider' }}
  >
    <ThemeScope theme="base">
      <Button
        size="small"
        variant="text"
        aria-expanded={expanded}
        aria-controls={controls}
        onClick={onClick}
      >
        {label}
      </Button>
    </ThemeScope>
  </Box>
);

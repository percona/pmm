import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Collapse from '@mui/material/Collapse';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';
import { StatusIcon } from '@percona/peak-ui';
import { type FC, type Ref, useEffect, useId, useRef, useState } from 'react';
import { capAttention } from '../hub.attention';
import { ATTENTION_SECTION_ID } from '../hub.doors';
import { Messages } from '../hub.messages';
import type {
  HubAlertCoverage,
  HubAttentionItem,
  HubDoor,
  HubSectionState,
} from '../hub.types';
import { DoorButton, DoorLink } from './DoorLink';
import { HubSection, SectionState } from './HubSection';
import { HUB_CONTAINER } from './hub.sx';
import { formatSince } from './hub.time';
import { LiveAge } from './LiveAge';

export const ATTENTION_CAP = 5;

export interface NeedsAttentionSectionProps {
  items: HubAttentionItem[];
  state: HubSectionState;
  door?: HubDoor;
  checkedAt: string;
  /** What no alert rule covers yet, beneath the rows. */
  coverage?: HubAlertCoverage;
  onRetry: () => void;
}

/** A line under the rows: Needs attention lists what alert rules catch, so it says what they cannot. */
const CoverageLine: FC<{ coverage: HubAlertCoverage }> = ({
  coverage: { text, door, detail },
}) => (
  <Box
    sx={{ display: 'flex', alignItems: 'flex-start', gap: 1 }}
    data-testid="hub-alert-coverage"
  >
    <StatusIcon size="small" severity="info" />
    <Typography variant="body2" color="text.secondary">
      {text}
      {door && (
        <>
          {' '}
          <DoorLink door={door} underline="always" />
        </>
      )}
      {detail && ` ${detail}`}
    </Typography>
  </Box>
);

const AttentionRow: FC<{
  item: HubAttentionItem;
  divider: boolean;
  /** Set on the first row Show more reveals, which then takes focus. */
  rowRef?: Ref<HTMLLIElement>;
}> = ({ item, divider, rowRef }) => {
  const secondary = [item.detail, item.where, formatSince(item.since)]
    .filter(Boolean)
    .join(' · ');

  return (
    <ListItem
      ref={rowRef}
      tabIndex={rowRef ? -1 : undefined}
      divider={divider}
      alignItems="flex-start"
    >
      <ListItemIcon>
        <StatusIcon
          size="small"
          severity={item.severity}
          titleAccess={Messages.attention.severity[item.severity]}
        />
      </ListItemIcon>
      <Box
        sx={(theme) => ({
          flex: 1,
          minWidth: 0,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'flex-start',
          gap: 0.5,
          [theme.containerQueries(HUB_CONTAINER).up('md')]: {
            flexDirection: 'row',
            alignItems: 'center',
            gap: 2,
          },
        })}
      >
        <ListItemText
          sx={{ my: 0, flex: 1, minWidth: 0 }}
          primary={item.title}
          secondary={secondary || undefined}
        />
        <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, flexShrink: 0 }}>
          <DoorButton door={item.door} />
          {item.secondaryDoor && <DoorButton door={item.secondaryDoor} />}
        </Box>
      </Box>
    </ListItem>
  );
};

/** Needs attention: every problem as a row with its doors, or one calm line. */
export const NeedsAttentionSection: FC<NeedsAttentionSectionProps> = ({
  items,
  state,
  door,
  checkedAt,
  coverage,
  onRetry,
}) => {
  const restId = useId();
  const [expanded, setExpanded] = useState(false);
  const { visible, hidden } = capAttention(items, ATTENTION_CAP);
  // The revealed rows sit above the button, so focus moves to them instead of past them.
  const firstRevealed = useRef<HTMLLIElement>(null);
  const revealing = useRef(false);

  useEffect(() => {
    if (expanded && revealing.current) {
      revealing.current = false;
      firstRevealed.current?.focus();
    }
  }, [expanded]);

  return (
    <HubSection
      id={ATTENTION_SECTION_ID}
      title={Messages.sections.attention}
      count={items.length ? items.length : undefined}
      countLabel={Messages.sectionCount.attention(items.length)}
      door={door}
    >
      <SectionState state={state} onRetry={onRetry} skeletonRows={2}>
        {items.length === 0 ? (
          // A source that could not be read leaves nothing to vouch for.
          state.status === 'error' ? null : (
            <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 1 }}>
              <StatusIcon size="small" severity="success" />
              <Typography variant="body2" color="text.secondary">
                <LiveAge at={checkedAt} format={Messages.attention.calm} />
              </Typography>
            </Box>
          )
        ) : (
          <Paper variant="outlined">
            <List disablePadding>
              {visible.map((item, index) => (
                <AttentionRow
                  key={item.id}
                  item={item}
                  divider={index < visible.length - 1 || hidden.length > 0}
                />
              ))}
            </List>
            {hidden.length > 0 && (
              <>
                <Collapse in={expanded} id={restId}>
                  <List disablePadding>
                    {hidden.map((item, index) => (
                      <AttentionRow
                        key={item.id}
                        item={item}
                        divider
                        rowRef={index === 0 ? firstRevealed : undefined}
                      />
                    ))}
                  </List>
                </Collapse>
                <Box sx={{ px: 1, py: 0.5 }}>
                  <Button
                    size="small"
                    variant="text"
                    aria-expanded={expanded}
                    aria-controls={restId}
                    onClick={() => {
                      revealing.current = !expanded;
                      setExpanded(!expanded);
                    }}
                  >
                    {expanded
                      ? Messages.attention.showFewer
                      : Messages.attention.showMore(hidden.length)}
                  </Button>
                </Box>
              </>
            )}
          </Paper>
        )}
        {coverage && (
          <Box sx={{ mt: 1.5 }}>
            <CoverageLine coverage={coverage} />
          </Box>
        )}
      </SectionState>
    </HubSection>
  );
};

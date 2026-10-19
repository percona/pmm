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
import { type FC, useId, useState } from 'react';
import { capAttention } from '../hub.attention';
import { Messages } from '../hub.messages';
import type { HubAttentionItem, HubDoor, HubSectionState } from '../hub.types';
import { DoorButton } from './DoorLink';
import { HubSection, SectionState } from './HubSection';
import { formatSince } from './hub.time';
import { LiveAge } from './LiveAge';

export const ATTENTION_CAP = 5;

export interface NeedsAttentionSectionProps {
  items: HubAttentionItem[];
  state: HubSectionState;
  door?: HubDoor;
  checkedAt: string;
  onRetry: () => void;
}

const AttentionRow: FC<{ item: HubAttentionItem; divider: boolean }> = ({
  item,
  divider,
}) => {
  const secondary = [item.detail, item.where, formatSince(item.since)]
    .filter(Boolean)
    .join(' · ');

  return (
    <ListItem divider={divider} alignItems="flex-start" sx={{ gap: 2 }}>
      <ListItemIcon sx={{ minWidth: 0 }}>
        <StatusIcon
          size="small"
          severity={item.severity}
          titleAccess={Messages.attention.severity[item.severity]}
        />
      </ListItemIcon>
      <Box
        sx={{
          flex: 1,
          minWidth: 0,
          display: 'flex',
          flexDirection: { xs: 'column', md: 'row' },
          alignItems: { xs: 'flex-start', md: 'center' },
          gap: { xs: 0.5, md: 2 },
        }}
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
  onRetry,
}) => {
  const restId = useId();
  const [expanded, setExpanded] = useState(false);
  const { visible, hidden } = capAttention(items, ATTENTION_CAP);

  return (
    <HubSection
      id="needs-attention"
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
                    {hidden.map((item) => (
                      <AttentionRow key={item.id} item={item} divider />
                    ))}
                  </List>
                </Collapse>
                <Box sx={{ px: 1, py: 0.5 }}>
                  <Button
                    size="small"
                    variant="text"
                    aria-expanded={expanded}
                    aria-controls={restId}
                    onClick={() => setExpanded((open) => !open)}
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
      </SectionState>
    </HubSection>
  );
};

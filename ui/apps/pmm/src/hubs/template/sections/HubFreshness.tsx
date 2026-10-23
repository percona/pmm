import RefreshOutlined from '@mui/icons-material/RefreshOutlined';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import { keyframes } from '@mui/material/styles';
import { StateIndicator, Tooltip } from '@percona/peak-ui';
import { type FC, useEffect, useRef, useState } from 'react';
import { TextSelect } from 'components/text-select';
import { Messages } from '../hub.messages';
import { HUB_REFRESHES } from '../hub.params';
import type { HubFreshness as Freshness, HubRefresh } from '../hub.types';
import { visuallyHidden } from './hub.sx';
import { LiveAge } from './LiveAge';

export interface HubFreshnessProps {
  freshness: Freshness;
  onRefresh: () => void;
  onIntervalChange: (interval: HubRefresh) => void;
}

const messages = Messages.freshness;

const OPTIONS = HUB_REFRESHES.map((value) => ({
  value,
  label: value === 'off' ? messages.offOption : value,
}));

/** One turn of the refresh icon, in ms. */
export const HUB_TURN_MS = 800;

const turn = keyframes`to { transform: rotate(360deg); }`;
const dim = keyframes`50% { opacity: 0.4; }`;

// With reduced motion it dims instead, at the same rhythm.
const spinning = {
  animation: `${turn} ${HUB_TURN_MS}ms linear infinite`,
  '@media (prefers-reduced-motion: reduce)': {
    animation: `${dim} ${HUB_TURN_MS}ms linear infinite`,
  },
};

// Disabled without losing focus or its tooltip, which the disabled attribute would take.
const busy = {
  '&[aria-disabled="true"]': {
    color: 'text.disabled',
    cursor: 'default',
    '--IconButton-hoverBg': 'transparent',
    '&:hover': { backgroundColor: 'transparent' },
  },
};

/** How current the data is: the auto-refresh interval and Refresh, or how long ago the data stopped once it falls behind. */
export const HubFreshness: FC<HubFreshnessProps> = ({
  freshness: { state, at, interval, refreshing },
  onRefresh,
  onIntervalChange,
}) => {
  // The update time when Refresh was pressed; a newer one means that refresh landed.
  const [refreshedFrom, setRefreshedFrom] = useState<number | null>(null);
  const [turning, setTurning] = useState(false);
  const turningFrom = useRef(0);
  const [seenAt, setSeenAt] = useState(at);
  const stale = state === 'stale';
  const refreshed =
    refreshedFrom !== null && at !== null && at > refreshedFrom && !stale;

  // Turns while a refresh runs, and once for any update that lands between renders.
  if (at !== seenAt) {
    setSeenAt(at);
    setTurning(true);
  }
  if (refreshing && !turning) {
    setTurning(true);
  }

  useEffect(() => {
    if (turning) {
      turningFrom.current = Date.now();
    }
  }, [turning]);

  // Stops at the end of a whole turn once nothing runs; a timer, since a hidden tab pauses the animation.
  useEffect(() => {
    if (!turning || refreshing) {
      return;
    }
    const elapsed = Date.now() - turningFrom.current;
    const id = setTimeout(
      () => setTurning(false),
      HUB_TURN_MS - (elapsed % HUB_TURN_MS)
    );

    return () => clearTimeout(id);
  }, [turning, refreshing]);

  const refresh = () => {
    if (turning) {
      return;
    }
    setTurning(true);
    setRefreshedFrom(at ?? 0);
    onRefresh();
  };

  const icon = (
    <RefreshOutlined
      data-turning={turning || undefined}
      sx={turning ? spinning : undefined}
    />
  );

  return (
    <Box
      sx={{
        display: 'flex',
        // A stale line that wraps keeps Retry level with its first line.
        alignItems: stale ? 'flex-start' : 'center',
        gap: stale ? 0.5 : 0,
      }}
      data-testid="hub-freshness"
      data-state={state}
    >
      {stale ? (
        <>
          <StateIndicator state="degraded" size="small" sx={{ minWidth: 0 }}>
            {at === null ? (
              messages.notYet
            ) : (
              <LiveAge at={at} format={messages.stale} coarse />
            )}
          </StateIndicator>
          <Button
            startIcon={icon}
            aria-disabled={turning || undefined}
            onClick={refresh}
            sx={{ ...busy, flexShrink: 0 }}
          >
            {messages.retry}
          </Button>
        </>
      ) : (
        <>
          {/* Neutral like Add, in Peak's Base theme; its menu stays in the PMM theme. */}
          <TextSelect
            value={interval}
            label={messages.selector}
            options={OPTIONS}
            selectedLabel={
              interval === 'off' ? messages.offSelected : undefined
            }
            onChange={onIntervalChange}
            buttonTheme="base"
            compactMenu
            buttonProps={{ size: 'medium', disableElevation: true }}
            data-testid-button="hub-auto-refresh"
          />
          <Tooltip title={messages.refresh}>
            <IconButton
              aria-label={messages.refresh}
              aria-disabled={turning || undefined}
              onClick={refresh}
              sx={busy}
            >
              {icon}
            </IconButton>
          </Tooltip>
        </>
      )}
      {/* Says when a refresh lands, and once when the data stops coming. */}
      <Box component="span" role="status" sx={visuallyHidden}>
        {stale ? messages.announceStale : refreshed ? messages.refreshed : ''}
      </Box>
    </Box>
  );
};

import RefreshOutlined from '@mui/icons-material/RefreshOutlined';
import Box from '@mui/material/Box';
import IconButton from '@mui/material/IconButton';
import Typography from '@mui/material/Typography';
import { StatusIcon, Tooltip } from '@percona/peak-ui';
import { type FC, useState } from 'react';
import { Messages } from '../hub.messages';
import type { HubFreshness as Freshness } from '../hub.types';
import { focusableText, visuallyHidden } from './hub.sx';
import { LiveAge } from './LiveAge';

export interface HubFreshnessProps {
  freshness: Freshness;
  onRefresh: () => void;
}

/** When the data was updated, with Refresh; a warning once a source misses its refreshes. */
export const HubFreshness: FC<HubFreshnessProps> = ({
  freshness: { at, stale },
  onRefresh,
}) => {
  // The update time when Refresh was pressed; a newer one means that refresh landed.
  const [refreshedFrom, setRefreshedFrom] = useState<number | null>(null);
  const refreshed = refreshedFrom !== null && at > refreshedFrom && !stale;
  const age = (
    <Typography
      variant="caption"
      color={stale ? 'warning.main' : 'text.secondary'}
      noWrap
      tabIndex={stale ? 0 : undefined}
      sx={stale ? focusableText : undefined}
    >
      <LiveAge at={at} format={Messages.updated} coarse />
    </Typography>
  );

  return (
    <Box
      sx={{ display: 'flex', alignItems: 'center', gap: 0.75 }}
      data-testid="hub-freshness"
    >
      {stale && (
        <StatusIcon
          size="small"
          severity="warning"
          titleAccess={Messages.attention.severity.warning}
        />
      )}
      {stale ? (
        <Tooltip title={Messages.stale} describeChild>
          {age}
        </Tooltip>
      ) : (
        age
      )}
      <Tooltip title={Messages.refresh}>
        <IconButton
          aria-label={Messages.refresh}
          onClick={() => {
            setRefreshedFrom(at);
            onRefresh();
          }}
          size="small"
          // Quiet in the note's secondary tone until pointed at or focused; the margin pulls its 12px padding toward the note.
          sx={{
            ml: -0.75,
            color: 'text.secondary',
            '&:hover, &.Mui-focusVisible': { color: 'text.primary' },
          }}
        >
          <RefreshOutlined fontSize="small" />
        </IconButton>
      </Tooltip>
      {/* Says when a refresh lands, and when the data turns stale. */}
      <Box component="span" role="status" sx={visuallyHidden}>
        {stale ? Messages.stale : refreshed ? Messages.refreshed : ''}
      </Box>
    </Box>
  );
};

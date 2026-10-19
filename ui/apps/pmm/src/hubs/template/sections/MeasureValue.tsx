import Box from '@mui/material/Box';
import { StatusIcon, Tooltip } from '@percona/peak-ui';
import { formatTimestamp } from '@pmm-extensions/framework';
import type { FC } from 'react';
import { Messages } from '../hub.messages';
import type { HubMeasure } from '../hub.types';
import { DoorLink } from './DoorLink';
import { focusableText, visuallyHidden } from './hub.sx';
import { formatDuration } from './hub.time';

export interface MeasureValueProps {
  measure?: HubMeasure;
}

/** A duration measure: plain when fine, with a StatusIcon when past a threshold. */
export const MeasureValue: FC<MeasureValueProps> = ({ measure }) => {
  if (!measure) {
    return null;
  }

  if (measure.kind === 'notApplicable') {
    return (
      <Box component="span" sx={{ color: 'text.secondary' }}>
        <span aria-hidden="true">—</span>
        <Box component="span" sx={visuallyHidden}>
          {measure.reason}
        </Box>
      </Box>
    );
  }

  if (measure.kind === 'noData') {
    const lastSeen = formatTimestamp(measure.lastSeen)?.title;
    const text = (
      <Box
        component="span"
        tabIndex={lastSeen ? 0 : undefined}
        sx={[{ color: 'text.secondary' }, !!lastSeen && focusableText]}
      >
        {Messages.measure.noData}
      </Box>
    );

    return lastSeen ? (
      <Tooltip title={Messages.measure.lastSeen(lastSeen)} describeChild>
        {text}
      </Tooltip>
    ) : (
      text
    );
  }

  const duration = formatDuration(measure.seconds);
  const label = (
    <>
      {duration}
      {measure.level !== 'ok' && (
        <Box component="span" sx={visuallyHidden}>
          {` ${Messages.measure.over[measure.level]}`}
        </Box>
      )}
    </>
  );
  const value = measure.door ? (
    <DoorLink door={measure.door} color="inherit">
      {label}
    </DoorLink>
  ) : (
    <Box
      component="span"
      tabIndex={measure.rule ? 0 : undefined}
      sx={measure.rule ? focusableText : undefined}
    >
      {label}
    </Box>
  );

  return (
    <Box
      component="span"
      sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5 }}
    >
      {measure.level !== 'ok' && (
        <StatusIcon size="small" severity={measure.level} />
      )}
      {measure.rule ? (
        <Tooltip title={measure.rule} describeChild>
          {value}
        </Tooltip>
      ) : (
        value
      )}
    </Box>
  );
};

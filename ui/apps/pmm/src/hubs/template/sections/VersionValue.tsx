import Box from '@mui/material/Box';
import { Chip, StatusIcon, Tooltip } from '@percona/peak-ui';
import type { FC } from 'react';
import { Messages } from '../hub.messages';
import type { HubUpdate, HubVersion } from '../hub.types';
import { focusableText, visuallyHidden } from './hub.sx';
import { formatAge } from './hub.time';

const ageOf = (at?: string) => (at ? formatAge(at) : undefined);

const updateNote = (update: HubUpdate): string => {
  switch (update.state) {
    case 'available':
      return Messages.version.available(
        update.latest,
        formatAge(update.checkedAt)
      );
    case 'upToDate':
      return Messages.version.upToDate(formatAge(update.checkedAt));
    default:
      return update.reason === 'failed'
        ? Messages.version.failed(ageOf(update.checkedAt))
        : Messages.version.stale(ageOf(update.checkedAt));
  }
};

export interface VersionValueProps {
  version?: HubVersion;
}

/** The version a node runs, a warning when its members differ, and what the update check says beneath it. */
export const VersionValue: FC<VersionValueProps> = ({ version }) => {
  if (!version) {
    return null;
  }
  const { effective, drift, update } = version;
  const notes = [
    ...(drift
      ? [
          Messages.version.drift(
            drift.map(({ version: each, members }) =>
              Messages.version.on(each, members)
            )
          ),
        ]
      : []),
    ...(update ? [updateNote(update)] : []),
  ];
  const content = (
    <Box
      component="span"
      tabIndex={notes.length ? 0 : undefined}
      sx={[
        {
          display: 'inline-flex',
          flexDirection: 'column',
          alignItems: 'flex-start',
          gap: 0.5,
          maxWidth: '100%',
        },
        notes.length > 0 && focusableText,
      ]}
    >
      <Box
        component="span"
        sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5 }}
      >
        {drift && <StatusIcon size="small" severity="warning" />}
        {effective}
        {drift && (
          <Box component="span" sx={visuallyHidden}>
            {` ${Messages.version.mixed}`}
          </Box>
        )}
      </Box>
      {update?.state === 'available' && (
        <Chip
          size="small"
          variant="outlined"
          color="info"
          label={Messages.version.updateAvailable}
        />
      )}
      {update?.state === 'unknown' && (
        <Chip
          size="small"
          variant="outlined"
          label={Messages.version.checkFailed}
        />
      )}
    </Box>
  );

  return notes.length ? (
    <Tooltip
      title={notes.map((note) => (
        <Box key={note} component="span" sx={{ display: 'block' }}>
          {note}
        </Box>
      ))}
      describeChild
    >
      {content}
    </Tooltip>
  ) : (
    content
  );
};

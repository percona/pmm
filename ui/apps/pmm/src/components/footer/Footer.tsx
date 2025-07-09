import { Stack, Typography } from '@mui/material';
import { useUpdates } from 'contexts/updates';
import { FC } from 'react';
import { formatCheckDate } from './Footer.utils';
import { Messages } from './Footer.messages';

export const Footer: FC = () => {
  const { inProgress, versionInfo } = useUpdates();

  if (!versionInfo) return null;

  return (
    <Stack direction="row" gap={2} data-testid="pmm-footer">
      <Typography variant="body2">
        {Messages.version(versionInfo.installed.version)}
      </Typography>
      <Typography variant="body2" color="text.disabled">
        {inProgress
          ? Messages.inProgress
          : Messages.checkedOn(formatCheckDate(versionInfo.lastCheck))}
      </Typography>
    </Stack>
  );
};

import { Stack, Typography } from '@mui/material';
import { useUpdates } from 'contexts/updates';
import { FC } from 'react';
import { formatCheckDate } from './Footer.utils';
import { Messages } from './Footer.messages';

export const Footer: FC = () => {
  const { inProgress, versionInfo } = useUpdates();

  if (!versionInfo) return null;

  const { lastCheck } = versionInfo;
  let checkStatus: string | null = null;

  if (inProgress) {
    checkStatus = Messages.inProgress;
  } else if (lastCheck) {
    checkStatus = Messages.checkedOn(formatCheckDate(lastCheck));
  }

  return (
    <Stack direction="row" gap={2} data-testid="pmm-footer">
      <Typography variant="body2">
        {Messages.version(versionInfo.installed.version)}
      </Typography>
      {checkStatus && (
        <Typography variant="body2" color="text.disabled">
          {checkStatus}
        </Typography>
      )}
    </Stack>
  );
};

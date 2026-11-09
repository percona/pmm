import { Divider, Stack, Typography } from '@mui/material';
import { useUpdates } from 'contexts/updates';
import { FC } from 'react';
import { getCheckStatus } from './Footer.utils';
import { Messages } from './Footer.messages';

export const Footer: FC = () => {
  const { inProgress, versionInfo } = useUpdates();

  if (!versionInfo) return null;

  const checkStatus = getCheckStatus(versionInfo, inProgress);

  return (
    <Stack gap={2} data-testid="pmm-footer">
      <Divider />
      <Stack direction="row" gap={2}>
        <Typography variant="body2">
          {Messages.version(versionInfo.installed.version)}
        </Typography>
        {checkStatus && (
          <Typography variant="body2" color="text.disabled">
            {checkStatus}
          </Typography>
        )}
      </Stack>
    </Stack>
  );
};

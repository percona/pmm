import { Link, Stack, Typography } from '@mui/material';
import { useUpdates } from 'contexts/updates';
import { FC } from 'react';
import { formatCheckDate } from './UpdateFooter.utils';
import { Messages } from './UpdateFooter.messages';

export const UpdateFooter: FC = () => {
  const { inProgress, versionInfo, recheck } = useUpdates();

  console.log({ versionInfo });
  if (!versionInfo) return null;

  return (
    <Stack direction="row" gap={2}>
      <Typography variant="body2">
        {Messages.version(versionInfo.installed.version)}
      </Typography>
      <Typography variant="body2" color="text.disabled">
        {inProgress
          ? Messages.inProgress
          : Messages.checkedOn(formatCheckDate(versionInfo.lastCheck))}
      </Typography>
      {!inProgress && (
        <Typography variant="body2">
          <Link onClick={recheck}>{Messages.checkNow}</Link>
        </Typography>
      )}
    </Stack>
  );
};

import LockOutlinedIcon from '@mui/icons-material/LockOutlined';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { useTimezone } from 'hooks/utils/useTimezone';
import type { FC } from 'react';
import type { Dump } from 'types/dump.types';
import { Messages } from '../../DumpPage.messages';
import { formatDataWindow, formatTimeRange } from '../../DumpPage.utils';

interface Props {
  dump: Dump;
}

export const DataWindow: FC<Props> = ({ dump }) => {
  const timezone = useTimezone();

  return (
    <Stack>
      <Typography variant="body2">
        {formatDataWindow(dump, timezone)}
      </Typography>
      <Stack direction="row" gap={1} alignItems="center">
        <Typography variant="caption" color="text.secondary">
          {formatTimeRange(dump)}
        </Typography>
        {dump.encrypted && (
          <Tooltip title={Messages.encryptedHint}>
            <Stack
              component="span"
              direction="row"
              gap={0.5}
              alignItems="center"
            >
              <LockOutlinedIcon fontSize="inherit" />
              <Typography variant="caption">{Messages.encrypted}</Typography>
            </Stack>
          </Tooltip>
        )}
      </Stack>
    </Stack>
  );
};

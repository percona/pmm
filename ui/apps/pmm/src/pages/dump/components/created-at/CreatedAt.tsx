import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { useTimezone } from 'hooks/utils/useTimezone';
import type { FC } from 'react';
import { formatDate, formatRelativeTime } from '../../DumpPage.utils';

interface Props {
  value: string;
}

export const CreatedAt: FC<Props> = ({ value }) => {
  const timezone = useTimezone();

  return (
    <Tooltip title={formatDate(value, timezone)}>
      <Typography variant="body2" component="span">
        {formatRelativeTime(value)}
      </Typography>
    </Tooltip>
  );
};

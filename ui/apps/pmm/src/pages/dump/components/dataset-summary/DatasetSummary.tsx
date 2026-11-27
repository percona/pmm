import Typography from '@mui/material/Typography';
import { useTimezone } from 'hooks/utils/useTimezone';
import type { FC } from 'react';
import type { Dump } from 'types/dump.types';
import { Messages } from '../../DumpPage.messages';
import { formatDataWindow } from '../../DumpPage.utils';

interface Props {
  dump: Dump;
}

export const DatasetSummary: FC<Props> = ({ dump }) => {
  const timezone = useTimezone();
  const services = dump.serviceNames.join(', ') || Messages.allServices;

  return (
    <Typography variant="body2">
      {`${formatDataWindow(dump, timezone)} · ${services}`}
    </Typography>
  );
};

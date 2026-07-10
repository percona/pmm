import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Icon } from 'components/icon';
import { FC } from 'react';
import { AlertSeverity } from 'types/alerting.types';
import { SEVERITY_ICON_MAP } from './AlertSeverityDetail.constants';
import { capitalize } from 'utils/text.utils';
import UnavailableText from 'components/unavailable-text';

interface Props {
  severity?: string;
}

const AlertSeverityDetail: FC<Props> = ({ severity }) => {
  if (!severity) {
    return <UnavailableText />;
  }

  const icon = SEVERITY_ICON_MAP[severity as AlertSeverity];

  return (
    <Stack direction="row" spacing={0.5} alignItems="center">
      {icon && <Icon name={icon} />}
      <Typography>{capitalize(severity)}</Typography>
    </Stack>
  );
};

export default AlertSeverityDetail;

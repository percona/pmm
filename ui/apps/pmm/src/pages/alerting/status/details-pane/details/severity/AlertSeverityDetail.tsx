import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Icon } from 'components/icon';
import { FC } from 'react';
import { AlertSeverity } from 'types/alerting.types';
import { SEVERITY_ICON_MAP } from './AlertSeverityDetail.constants';
import { capitalize } from 'utils/text.utils';

interface Props {
  severity: AlertSeverity;
}

const AlertSeverityDetail: FC<Props> = ({ severity }) => (
  <Stack direction="row" spacing={0.5} alignItems="center">
    {SEVERITY_ICON_MAP[severity] && <Icon name={SEVERITY_ICON_MAP[severity]} />}
    <Typography>{capitalize(severity)}</Typography>
  </Stack>
);

export default AlertSeverityDetail;

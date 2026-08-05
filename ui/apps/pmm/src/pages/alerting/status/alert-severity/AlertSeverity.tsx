import Typography from '@mui/material/Typography';
import { FC } from 'react';
import type { AlertSeverity as Severity } from 'types/alerting.types';
import { capitalize } from 'utils/text.utils';
import { SEVERITY_ICON_MAP } from './AlertSeverity.constants';
import Stack from '@mui/material/Stack';

interface Props {
  severity: Severity;
}

const AlertSeverity: FC<Props> = ({ severity }) => (
  <Stack direction="row" alignItems="center" gap={0.55}>
    {SEVERITY_ICON_MAP[severity]}
    <Typography>{capitalize(severity)}</Typography>
  </Stack>
);

export default AlertSeverity;

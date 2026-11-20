import Typography from '@mui/material/Typography';
import { FC } from 'react';
import { capitalize } from 'utils/text.utils';
import { SEVERITY_BAND_CONFIG } from './AlertSeverity.constants';
import { getSeverityBand } from './AlertSeverity.utils';
import Stack from '@mui/material/Stack';

interface Props {
  severity?: string;
}

const AlertSeverity: FC<Props> = ({ severity }) => {
  const band = getSeverityBand(severity);

  if (!band) {
    return <Typography>{capitalize(severity ?? '')}</Typography>;
  }

  const { label, icon } = SEVERITY_BAND_CONFIG[band];
  const isCritical = band === 'critical';

  return (
    <Stack
      direction="row"
      alignItems="center"
      gap={0.55}
      data-testid={`alert-severity-${band}`}
    >
      {icon}
      <Typography
        color={isCritical ? 'error' : undefined}
        fontWeight={isCritical ? 'bold' : undefined}
      >
        {label}
      </Typography>
    </Stack>
  );
};

export default AlertSeverity;

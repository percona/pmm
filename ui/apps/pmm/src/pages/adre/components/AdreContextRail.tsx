import { Stack, Typography } from '@mui/material';
import { FC } from 'react';
import { AdreStepsList } from 'components/adre/AdreStepsList';
import type { ProgressStep } from 'hooks/useAdreChat';
import { AdreAlertsPanel, type AlertItem } from './AdreAlertsPanel';

export interface AdreContextRailProps {
  alerts: AlertItem[];
  /** The latest answer's tool steps, live while it is written. */
  steps: ProgressStep[];
  working: boolean;
}

const heading = {
  fontFamily: 'Poppins, sans-serif',
  fontSize: 14,
  fontWeight: 600,
} as const;

/** The side panel of the Assistant page: firing alerts, then how the latest answer was made. */
export const AdreContextRail: FC<AdreContextRailProps> = ({
  alerts,
  steps,
  working,
}) => {
  return (
    <Stack gap={3.5}>
      {alerts.length > 0 && <AdreAlertsPanel alerts={alerts} />}
      {steps.length > 0 && (
        <Stack
          component="section"
          aria-label="How this answer was made"
          gap={1.5}
        >
          <Typography component="h2" sx={heading}>
            How this answer was made
          </Typography>
          <AdreStepsList steps={steps} working={working} />
        </Stack>
      )}
      {alerts.length === 0 && steps.length === 0 && (
        <Typography sx={{ fontSize: 13, color: '#8E8C95' }}>
          The steps of the latest answer show here.
        </Typography>
      )}
    </Stack>
  );
};

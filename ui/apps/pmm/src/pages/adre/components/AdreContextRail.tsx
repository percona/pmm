import Check from '@mui/icons-material/Check';
import { Box, CircularProgress, Stack, Typography } from '@mui/material';
import { FC } from 'react';
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
          <Stack
            component="ol"
            gap={1.5}
            sx={{ m: 0, p: 0, listStyle: 'none' }}
          >
            {steps.map((step) => (
              <Stack component="li" key={step.id} direction="row" gap={1.25}>
                <Box sx={{ flex: '0 0 16px', mt: '2px', display: 'flex' }}>
                  {step.status === 'running' && working ? (
                    <CircularProgress size={14} aria-label="Running" />
                  ) : (
                    <Check
                      aria-label="Done"
                      sx={{ fontSize: 16, color: '#86EFAC' }}
                    />
                  )}
                </Box>
                <Stack gap={0.25} sx={{ minWidth: 0 }}>
                  <Typography
                    sx={{
                      fontFamily: 'Roboto Mono, monospace',
                      fontSize: 12,
                      color: '#D9D7DE',
                      overflowWrap: 'anywhere',
                    }}
                  >
                    {step.toolName}
                  </Typography>
                  {step.description && (
                    <Typography
                      sx={{
                        fontSize: 12,
                        color: '#A9A7B0',
                        overflowWrap: 'anywhere',
                        display: '-webkit-box',
                        WebkitLineClamp: 3,
                        WebkitBoxOrient: 'vertical',
                        overflow: 'hidden',
                      }}
                      title={step.description}
                    >
                      {step.description}
                    </Typography>
                  )}
                </Stack>
              </Stack>
            ))}
          </Stack>
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

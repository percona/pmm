import Check from '@mui/icons-material/Check';
import { Box, CircularProgress, Stack, Typography } from '@mui/material';
import { FC } from 'react';
import type { ProgressStep } from 'hooks/useAdreChat';
import {
  formatTokenCount,
  formatUsdCost,
  type HolmesUsageDisplay,
} from 'utils/holmesUsageFormat';
import { AdreAlertsPanel, type AlertItem } from './AdreAlertsPanel';

export interface AdreContextRailProps {
  alerts: AlertItem[];
  /** The latest answer's tool steps, live while it is written. */
  steps: ProgressStep[];
  working: boolean;
  usage?: HolmesUsageDisplay;
}

const heading = {
  fontFamily: 'Poppins, sans-serif',
  fontSize: 14,
  fontWeight: 600,
} as const;

/** The side panel of the Assistant page: firing alerts, then how the latest answer was made and what it cost. */
export const AdreContextRail: FC<AdreContextRailProps> = ({
  alerts,
  steps,
  working,
  usage,
}) => {
  const tiles = [
    ['Model', usage?.model],
    ['Cost', formatUsdCost(usage?.totalCost)],
    ['Tokens', formatTokenCount(usage?.totalTokens)],
    ['Cached', formatTokenCount(usage?.cachedTokens)],
  ].filter((t): t is [string, string] => !!t[1]);

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
      {tiles.length > 0 && (
        <Box
          component="section"
          aria-label="Usage"
          sx={{
            display: 'grid',
            gridTemplateColumns: 'repeat(2, minmax(0, 1fr))',
            gap: 1,
          }}
        >
          {tiles.map(([name, value]) => (
            <Box
              key={name}
              sx={{
                px: 1.5,
                py: 1.25,
                borderRadius: '10px',
                bgcolor: '#1F1E23',
              }}
            >
              <Typography sx={{ fontSize: 11, color: '#8E8C95' }}>
                {name}
              </Typography>
              <Typography
                sx={{
                  mt: 0.25,
                  fontSize: 14,
                  fontWeight: 500,
                  overflowWrap: 'anywhere',
                }}
              >
                {value}
              </Typography>
            </Box>
          ))}
        </Box>
      )}
      {alerts.length === 0 && steps.length === 0 && tiles.length === 0 && (
        <Typography sx={{ fontSize: 13, color: '#8E8C95' }}>
          The steps and usage of the latest answer show here.
        </Typography>
      )}
    </Stack>
  );
};

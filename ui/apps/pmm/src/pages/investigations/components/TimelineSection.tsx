import { Box, Stack, Typography } from '@mui/material';
import { FC } from 'react';
import type { InvestigationTimelineEvent } from 'api/investigations';
import { adreColors, headingFont } from 'components/adre/adreAnswerSx';

const formatEventTime = (eventTime: string) => {
  const d = new Date(eventTime);
  if (Number.isNaN(d.getTime())) return '';
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getUTCFullYear()}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())} ${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}:${pad(d.getUTCSeconds())}`;
};

/** What happened when, one event per row along a rail. */
export const TimelineSection: FC<{ events: InvestigationTimelineEvent[] }> = ({
  events,
}) => {
  if (!events || events.length === 0) return null;

  return (
    <Box component="section" aria-label="Timeline">
      <Typography
        component="h2"
        sx={{
          fontFamily: headingFont,
          fontSize: 18,
          fontWeight: 600,
          color: adreColors.text,
          mb: 1.5,
        }}
      >
        Timeline
        <Box
          component="span"
          sx={{ ml: 1, fontSize: 12, fontWeight: 400, color: adreColors.muted }}
        >
          UTC
        </Box>
      </Typography>
      <Stack component="ol" sx={{ m: 0, p: 0, listStyle: 'none' }}>
        {events.map((event, i) => {
          const eventTime =
            event.eventTime ??
            (event as { event_time?: string }).event_time ??
            '';
          const last = i === events.length - 1;
          return (
            <Box
              component="li"
              key={event.id}
              sx={{
                display: 'grid',
                gridTemplateColumns: {
                  xs: '1fr',
                  sm: '150px 20px minmax(0, 1fr)',
                },
                columnGap: 1.5,
              }}
            >
              <Typography
                sx={{
                  fontFamily: 'Roboto Mono, monospace',
                  fontSize: 12,
                  color: adreColors.secondary,
                  pt: '2px',
                }}
              >
                {formatEventTime(eventTime)}
              </Typography>
              <Box
                aria-hidden="true"
                sx={{
                  display: { xs: 'none', sm: 'flex' },
                  flexDirection: 'column',
                  alignItems: 'center',
                }}
              >
                <Box
                  sx={{
                    mt: '6px',
                    width: 9,
                    height: 9,
                    borderRadius: '50%',
                    bgcolor: adreColors.accent,
                    flexShrink: 0,
                  }}
                />
                {!last && (
                  <Box
                    sx={{ flex: 1, width: '1px', bgcolor: adreColors.border }}
                  />
                )}
              </Box>
              <Box sx={{ pb: last ? 0 : 2, minWidth: 0 }}>
                <Typography
                  sx={{ fontSize: 14, fontWeight: 500, color: adreColors.text }}
                >
                  {event.title}
                </Typography>
                {event.description && (
                  <Typography
                    sx={{
                      fontSize: 13,
                      color: adreColors.secondary,
                      overflowWrap: 'anywhere',
                    }}
                  >
                    {event.description}
                  </Typography>
                )}
              </Box>
            </Box>
          );
        })}
      </Stack>
    </Box>
  );
};

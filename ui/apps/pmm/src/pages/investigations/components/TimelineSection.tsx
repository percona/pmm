import { Box, Typography } from '@mui/material';
import { FC } from 'react';
import type { InvestigationTimelineEvent } from 'api/investigations';

export const TimelineSection: FC<{ events: InvestigationTimelineEvent[] }> = ({
  events,
}) => {
  if (!events || events.length === 0) return null;

  const formatEventTime = (eventTime: string) => {
    try {
      const d = new Date(eventTime);
      const y = d.getUTCFullYear();
      const m = String(d.getUTCMonth() + 1).padStart(2, '0');
      const day = String(d.getUTCDate()).padStart(2, '0');
      const h = String(d.getUTCHours()).padStart(2, '0');
      const min = String(d.getUTCMinutes()).padStart(2, '0');
      const sec = String(d.getUTCSeconds()).padStart(2, '0');
      return `${y}-${m}-${day} ${h}:${min}:${sec} UTC`;
    } catch {
      return '';
    }
  };

  return (
    <Box sx={{ mb: 2 }}>
      <Typography variant="h6" sx={{ mb: 1 }}>
        Timeline
      </Typography>
      <Box
        sx={{
          position: 'relative',
          pl: 2,
          borderLeft: '2px solid',
          borderColor: 'primary.main',
        }}
      >
        {events.map((event) => {
          const eventTime = event.eventTime ?? (event as { event_time?: string }).event_time ?? '';
          const timeStr = formatEventTime(eventTime);
          const label = [timeStr, event.title, event.description]
            .filter(Boolean)
            .join(' - ');
          return (
            <Box
              key={event.id}
              sx={{
                position: 'relative',
                mb: 1.5,
                '&::before': {
                  content: '""',
                  position: 'absolute',
                  left: -9,
                  top: 6,
                  width: 8,
                  height: 8,
                  borderRadius: '50%',
                  bgcolor: 'primary.main',
                },
              }}
            >
              <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap' }}>
                {label}
              </Typography>
            </Box>
          );
        })}
      </Box>
    </Box>
  );
};

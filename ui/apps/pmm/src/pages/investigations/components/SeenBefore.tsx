import {
  Box,
  Card,
  CardContent,
  Chip,
  Link,
  Stack,
  Typography,
} from '@mui/material';
import { FC } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import type { SimilarInvestigation } from 'api/investigations';
import { PMM_NEW_NAV_PATH } from 'lib/constants';
import { formatSeenDate } from './seenBefore.utils';

const clampSx = {
  display: '-webkit-box',
  WebkitLineClamp: 2,
  WebkitBoxOrient: 'vertical',
  overflow: 'hidden',
} as const;

/** Past investigations that look like this incident: why they match and how they ended. */
export const SeenBefore: FC<{ items: SimilarInvestigation[] }> = ({
  items,
}) => {
  if (!items.length) return null;
  return (
    <Box component="section" aria-label="Seen before" sx={{ mb: 2 }}>
      <Typography variant="h6" sx={{ mb: 1 }}>
        Seen before
      </Typography>
      <Stack gap={1}>
        {items.map((s) => {
          const outcome = [
            s.rootCauseSummary
              ? ['Root cause', s.rootCauseSummary]
              : ['Summary', s.summary],
            ['Resolution', s.resolutionSummary],
          ].filter(([, text]) => !!text);
          return (
            <Card key={s.id} variant="outlined">
              <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
                <Stack
                  direction="row"
                  alignItems="center"
                  flexWrap="wrap"
                  gap={1}
                  sx={{ mb: outcome.length ? 0.75 : 0 }}
                >
                  <Link
                    component={RouterLink}
                    to={`${PMM_NEW_NAV_PATH}/investigations/${s.id}`}
                    sx={{ fontWeight: 500 }}
                  >
                    {s.title}
                  </Link>
                  <Typography variant="body2" color="text.secondary">
                    {formatSeenDate(s.createdAt)} · {s.status}
                  </Typography>
                  {s.reasons.map((r) => (
                    <Chip key={r} size="small" label={r} variant="outlined" />
                  ))}
                </Stack>
                {outcome.map(([label, text]) => (
                  <Typography
                    key={label}
                    variant="body2"
                    color="text.secondary"
                    sx={clampSx}
                    title={text}
                  >
                    <Box component="strong" sx={{ color: 'text.primary' }}>
                      {label}:
                    </Box>{' '}
                    {text}
                  </Typography>
                ))}
              </CardContent>
            </Card>
          );
        })}
      </Stack>
    </Box>
  );
};

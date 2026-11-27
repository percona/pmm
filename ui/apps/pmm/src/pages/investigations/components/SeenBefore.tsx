import { Box, Link, Stack, Typography } from '@mui/material';
import { FC } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import type { SimilarInvestigation } from 'api/investigations';
import { adreColors, headingFont } from 'components/adre/adreAnswerSx';
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
    <Box component="section" aria-label="Seen before">
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
            <Box
              key={s.id}
              sx={{
                p: 1.75,
                borderRadius: '12px',
                bgcolor: adreColors.card,
                border: `1px solid ${adreColors.border}`,
              }}
            >
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
                  sx={{ fontWeight: 500, color: adreColors.accentText }}
                >
                  {s.title}
                </Link>
                <Typography sx={{ fontSize: 12, color: adreColors.muted }}>
                  {formatSeenDate(s.createdAt)} · {s.status}
                </Typography>
                {s.reasons.map((r) => (
                  <Box
                    key={r}
                    component="span"
                    sx={{
                      px: 1,
                      borderRadius: 999,
                      fontSize: 12,
                      color: adreColors.secondary,
                      border: `1px solid ${adreColors.border}`,
                    }}
                  >
                    {r}
                  </Box>
                ))}
              </Stack>
              {outcome.map(([label, text]) => (
                <Typography
                  key={label}
                  title={text}
                  sx={{ ...clampSx, fontSize: 13, color: adreColors.secondary }}
                >
                  <Box component="strong" sx={{ color: adreColors.text }}>
                    {label}:
                  </Box>{' '}
                  {text}
                </Typography>
              ))}
            </Box>
          );
        })}
      </Stack>
    </Box>
  );
};

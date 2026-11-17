import { Box, Button, Collapse, Stack, Typography } from '@mui/material';
import { FC, useState } from 'react';
import type { Investigation } from 'api/investigations';
import { adreColors, headingFont } from 'components/adre/adreAnswerSx';

const CONFIDENCE_COLORS: Record<string, string> = {
  high: '#86EFAC',
  medium: '#FCD34D',
  low: '#FCA5A5',
};

const labelSx = {
  fontSize: 12,
  fontWeight: 500,
  color: adreColors.accentText,
  mb: 0.5,
} as const;

const textSx = {
  fontSize: 15,
  lineHeight: 1.65,
  color: adreColors.body,
  whiteSpace: 'pre-wrap',
  overflowWrap: 'anywhere',
} as const;

export interface InvestigationLeadProps {
  inv: Investigation;
  /** The summary to show; empty when it only repeats the request. */
  summary: string;
  /** The longer summary, shown when there are no report sections to carry it. */
  detail?: string;
  running: boolean;
}

/** What the investigation found, first: summary, root cause, resolution and how sure it is. */
export const InvestigationLead: FC<InvestigationLeadProps> = ({
  inv,
  summary,
  detail,
  running,
}) => {
  const [showEvidence, setShowEvidence] = useState(false);
  const evidence = inv.evidence ?? [];
  const rootCause = inv.rootCauseSummary?.trim();
  const resolution = inv.resolutionSummary?.trim();
  const hasFindings = !!(summary.trim() || rootCause || resolution || detail);
  const hasConfidence = (inv.confidenceScore ?? 0) > 0 || evidence.length > 0;
  const band = (inv.confidence || 'medium').toLowerCase();

  if (!hasFindings) {
    return (
      <Box
        component="section"
        aria-label="Summary"
        sx={{
          p: 2.25,
          borderRadius: '16px',
          border: `1px dashed ${adreColors.border}`,
          color: adreColors.secondary,
          fontSize: 14,
        }}
      >
        {running
          ? 'The investigation is running. Its summary, root cause and resolution show here when it finishes.'
          : 'No report yet. Run the investigation to get a summary, root cause and resolution.'}
      </Box>
    );
  }

  return (
    <Box
      component="section"
      aria-label="Summary"
      sx={{
        p: { xs: 2, md: 2.75 },
        borderRadius: '16px',
        bgcolor: adreColors.leadBg,
        border: `1px solid ${adreColors.leadBorder}`,
      }}
    >
      <Stack
        direction="row"
        alignItems="center"
        justifyContent="space-between"
        flexWrap="wrap"
        gap={1.5}
        sx={{ mb: 1.25 }}
      >
        <Typography
          component="h2"
          sx={{ fontFamily: headingFont, fontSize: 17, fontWeight: 600 }}
        >
          Summary
        </Typography>
        {hasConfidence && (
          <Stack direction="row" alignItems="center" gap={1}>
            <Box
              component="span"
              title={inv.confidenceRationale || undefined}
              sx={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: 0.75,
                px: 1.25,
                py: 0.25,
                borderRadius: 999,
                bgcolor: 'rgba(0,0,0,0.25)',
                fontSize: 12,
                color: CONFIDENCE_COLORS[band] ?? adreColors.secondary,
              }}
            >
              <Box
                component="span"
                sx={{
                  width: 7,
                  height: 7,
                  borderRadius: '50%',
                  bgcolor: 'currentColor',
                }}
              />
              {`${band.charAt(0).toUpperCase()}${band.slice(1)} confidence · ${inv.confidenceScore ?? 0}`}
            </Box>
            <Button
              size="small"
              onClick={() => setShowEvidence((v) => !v)}
              aria-expanded={showEvidence}
              sx={{ color: adreColors.accentText, minWidth: 0 }}
            >
              {showEvidence ? 'Hide evidence' : `Evidence (${evidence.length})`}
            </Button>
          </Stack>
        )}
      </Stack>
      {summary.trim() && <Typography sx={textSx}>{summary}</Typography>}
      {detail && (
        <Typography sx={{ ...textSx, mt: 1.5, fontSize: 14 }}>
          {detail}
        </Typography>
      )}
      {(rootCause || resolution) && (
        <Box
          sx={{
            mt: summary.trim() || detail ? 2.25 : 0,
            display: 'grid',
            gridTemplateColumns: {
              xs: 'minmax(0, 1fr)',
              md: rootCause && resolution ? 'repeat(2, minmax(0, 1fr))' : '1fr',
            },
            gap: 2.5,
          }}
        >
          {rootCause && (
            <Box>
              <Typography sx={labelSx}>Root cause</Typography>
              <Typography sx={{ ...textSx, fontSize: 14 }}>
                {rootCause}
              </Typography>
            </Box>
          )}
          {resolution && (
            <Box>
              <Typography sx={labelSx}>Resolution</Typography>
              <Typography sx={{ ...textSx, fontSize: 14 }}>
                {resolution}
              </Typography>
            </Box>
          )}
        </Box>
      )}
      <Collapse in={showEvidence} unmountOnExit>
        <Stack gap={1} sx={{ mt: 2.25 }}>
          {inv.confidenceRationale && (
            <Typography sx={{ fontSize: 13, color: adreColors.secondary }}>
              {inv.confidenceRationale}
            </Typography>
          )}
          {evidence.length === 0 ? (
            <Typography sx={{ fontSize: 13, color: adreColors.muted }}>
              No evidence entries.
            </Typography>
          ) : (
            evidence.map((e, idx) => (
              <Box
                key={e.id || idx}
                sx={{
                  p: 1.5,
                  borderRadius: '12px',
                  bgcolor: adreColors.card,
                  border: `1px solid ${adreColors.border}`,
                }}
              >
                <Typography sx={{ fontSize: 14, fontWeight: 500 }}>
                  {e.claim || 'Claim'}
                </Typography>
                <Typography sx={{ fontSize: 12, color: adreColors.muted }}>
                  {[e.kind, e.source_tool, e.source_ref]
                    .filter(Boolean)
                    .join(' · ')}
                </Typography>
                {e.excerpt && (
                  <Typography
                    sx={{
                      mt: 0.75,
                      fontFamily: 'Roboto Mono, monospace',
                      fontSize: 12,
                      color: adreColors.secondary,
                      whiteSpace: 'pre-wrap',
                      overflowWrap: 'anywhere',
                    }}
                  >
                    {e.excerpt}
                  </Typography>
                )}
              </Box>
            ))
          )}
        </Stack>
      </Collapse>
    </Box>
  );
};

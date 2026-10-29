import { Box, Typography } from '@mui/material';
import { FC, useMemo } from 'react';
import { adreColors } from './adreAnswerSx';
import {
  type ChartSpec,
  chartEmbedUrl,
  chartExploreQuery,
  rangeMs,
} from './adreChart.utils';
import { explore } from './adreToolSteps';
import { EmbedLink, GrafanaPanelEmbed } from './GrafanaPanelEmbed';

const pad = (n: number) => String(n).padStart(2, '0');
const utcDay = (d: Date) =>
  `${d.getUTCFullYear()}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())}`;
const utcTime = (d: Date) =>
  `${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}`;

/** The window in UTC, with the day once when both ends share it. */
const formatRange = ({ from, to }: { from: number; to: number }) => {
  const f = new Date(from);
  const t = new Date(to);
  return utcDay(f) === utcDay(t)
    ? `${utcDay(f)} ${utcTime(f)}–${utcTime(t)} UTC`
    : `${utcDay(f)} ${utcTime(f)} – ${utcDay(t)} ${utcTime(t)} UTC`;
};

/** A chart the Assistant drew from a ```chart block, as a live Grafana panel; a relative window is fixed when it first shows. */
const AdreChart: FC<{ spec: ChartSpec }> = ({ spec }) => {
  const range = useMemo(
    () => rangeMs(spec.from, spec.to),
    [spec.from, spec.to]
  );

  if (!range) {
    return (
      <Box sx={{ my: 1.5, fontSize: 13 }}>
        <Typography sx={{ fontSize: 13, color: '#FCA5A5' }}>
          {`Can't read the time range ${spec.from} – ${spec.to}`}
        </Typography>
        <Box
          component="pre"
          sx={{
            m: 0,
            mt: 1,
            p: 1,
            borderRadius: '8px',
            bgcolor: '#0D0C0F',
            color: adreColors.secondary,
            fontFamily: 'Roboto Mono, monospace',
            fontSize: 12,
            whiteSpace: 'pre-wrap',
            overflowWrap: 'anywhere',
          }}
        >
          {spec.query}
        </Box>
      </Box>
    );
  }

  return (
    <GrafanaPanelEmbed
      src={chartEmbedUrl(spec, range)}
      title={spec.title ?? 'Chart'}
      query={spec.query}
      caption={`${formatRange(range)} · ${spec.ds}`}
      link={
        <EmbedLink
          to={explore(spec.ds, chartExploreQuery(spec), {
            from: String(range.from),
            to: String(range.to),
          })}
        >
          Open in Explore
        </EmbedLink>
      }
    />
  );
};

export default AdreChart;

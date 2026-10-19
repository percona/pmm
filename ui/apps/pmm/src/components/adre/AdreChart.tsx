import { Box, Link, Skeleton, Stack, Typography } from '@mui/material';
import { LineChart } from '@mui/x-charts/LineChart';
import { FC } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { datasourceQuery } from 'api/grafanaQuery';
import { useAdreChart } from 'hooks/api/useAdreChart';
import { adreColors, headingFont } from './adreAnswerSx';
import { type ChartSpec, formatValue } from './adreChart.utils';
import { explore } from './adreToolSteps';

const HEIGHT = 240;
const COLORS = [
  '#B6B2FF',
  '#7DD3FC',
  '#86EFAC',
  '#FCD34D',
  '#FCA5A5',
  '#F0ABFC',
  '#5EEAD4',
  '#FDBA74',
  '#A5B4FC',
  '#D9F99D',
];

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

/** A chart the Assistant drew from a ```chart block: the query runs live through Grafana as the viewer. */
const AdreChart: FC<{ spec: ChartSpec }> = ({ spec }) => {
  const { range, ds, error, loading, data } = useAdreChart(spec);
  const type =
    ds?.type ??
    (spec.ds.startsWith('ClickHouse')
      ? 'grafana-clickhouse-datasource'
      : 'prometheus');
  const exploreQuery = datasourceQuery(type, spec.query);
  const exploreTo =
    range && exploreQuery
      ? explore(spec.ds, exploreQuery, {
          from: String(range.from),
          to: String(range.to),
        })
      : undefined;
  const empty = data && data.series.length === 0;

  return (
    <Box
      component="figure"
      aria-label={spec.title ?? 'Chart'}
      sx={{
        m: 0,
        my: 1.5,
        p: 1.75,
        borderRadius: '12px',
        bgcolor: adreColors.card,
        border: `1px solid ${adreColors.border}`,
        minWidth: 0,
      }}
    >
      <Stack
        component="figcaption"
        direction="row"
        flexWrap="wrap"
        alignItems="baseline"
        justifyContent="space-between"
        columnGap={2}
        sx={{ mb: 1 }}
      >
        <Box sx={{ minWidth: 0 }}>
          <Typography
            sx={{
              fontFamily: headingFont,
              fontSize: 14,
              fontWeight: 600,
              color: adreColors.text,
            }}
          >
            {spec.title ?? 'Chart'}
          </Typography>
          <Typography sx={{ fontSize: 12, color: adreColors.muted }}>
            {[range && formatRange(range), spec.ds].filter(Boolean).join(' · ')}
          </Typography>
        </Box>
        {exploreTo && (
          <Link
            component={RouterLink}
            to={exploreTo}
            sx={{ fontSize: 12, color: adreColors.accentText }}
          >
            Open in Explore
          </Link>
        )}
      </Stack>

      {loading && <Skeleton variant="rounded" height={HEIGHT} />}
      {(error || empty) && (
        <Box sx={{ fontSize: 13, color: adreColors.secondary }}>
          <Typography
            sx={{
              fontSize: 13,
              color: error ? '#FCA5A5' : adreColors.secondary,
            }}
          >
            {error ?? 'The query returned no data for this window.'}
          </Typography>
          <Box
            component="pre"
            sx={{
              m: 0,
              mt: 1,
              p: 1,
              borderRadius: '8px',
              bgcolor: '#0D0C0F',
              fontFamily: 'Roboto Mono, monospace',
              fontSize: 12,
              whiteSpace: 'pre-wrap',
              overflowWrap: 'anywhere',
            }}
          >
            {spec.query}
          </Box>
        </Box>
      )}
      {data && !empty && (
        <>
          <LineChart
            height={HEIGHT}
            skipAnimation
            colors={COLORS}
            grid={{ horizontal: true }}
            margin={{ left: 8, right: 16, top: 8, bottom: 8 }}
            xAxis={[
              {
                data: data.times.map((t) => new Date(t)),
                scaleType: 'time',
                valueFormatter: (d: Date) => utcTime(d),
              },
            ]}
            yAxis={[
              { valueFormatter: (v: number) => formatValue(v, spec.unit) },
            ]}
            series={data.series.map((s) => ({
              data: s.data,
              label: s.label,
              showMark: false,
              valueFormatter: (v: number | null) => formatValue(v, spec.unit),
            }))}
            sx={{
              '& .MuiChartsAxis-tickLabel': {
                fill: `${adreColors.secondary} !important`,
              },
              '& .MuiChartsAxis-line, & .MuiChartsAxis-tick': {
                stroke: `${adreColors.border} !important`,
              },
              '& .MuiChartsGrid-line': { stroke: adreColors.divider },
              '& .MuiChartsLegend-label': { color: adreColors.secondary },
            }}
          />
          {data.hidden > 0 && (
            <Typography sx={{ mt: 0.5, fontSize: 12, color: adreColors.muted }}>
              {data.hidden} smaller series not shown
            </Typography>
          )}
        </>
      )}
    </Box>
  );
};

export default AdreChart;

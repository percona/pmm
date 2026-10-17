import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import {
  Box,
  Button,
  Paper,
  Stack,
  Tooltip,
  Typography,
  useTheme,
} from '@mui/material';
import { FC, useEffect, useState } from 'react';
import { SeverityClass, VolumeBucket } from 'api/logs';
import { severityColor } from '../logsTheme';

interface LogsVolumeChartProps {
  buckets: VolumeBucket[];
  /** Bucket width as a protobuf Duration, e.g. "60s". */
  step: string;
  onZoom: (fromMs: number, toMs: number) => void;
  /** Returns to the range before the last zoom; the button shows only when set. */
  onBack?: () => void;
}

const CLASSES: { key: SeverityClass; label: string }[] = [
  { key: 'error', label: 'Error' },
  { key: 'warning', label: 'Warning' },
  { key: 'info', label: 'Info' },
  { key: 'other', label: 'Other' },
];

const HEIGHT = 120;
const AXIS_WIDTH = 40;
const MAX_TIME_LABELS = 6;
const DAY_MS = 24 * 60 * 60_000;

const formatTime = (ms: number) =>
  new Date(ms).toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });

const formatClock = (ms: number) =>
  new Date(ms).toLocaleTimeString(undefined, {
    hour: '2-digit',
    minute: '2-digit',
  });

const formatStep = (ms: number) => {
  if (ms < 60_000) return `${ms / 1000} s`;
  if (ms < 3_600_000) return `${ms / 60_000} min`;
  if (ms < DAY_MS) return `${ms / 3_600_000} h`;
  return `${ms / DAY_MS} d`;
};

const compact = new Intl.NumberFormat(undefined, { notation: 'compact' });

/** The smallest of 1, 2, 4, 6, 8 × 10^k at least n and at least 2, so the midline is a whole count. */
const roundUp = (n: number) => {
  const magnitude = 10 ** Math.floor(Math.log10(Math.max(2, n)));
  const step = [1, 2, 4, 6, 8, 10].find((s) => s * magnitude >= n) ?? 10;
  return Math.max(2, step * magnitude);
};

/** Stacked log counts per bucket; dragging across bars zooms to them. */
export const LogsVolumeChart: FC<LogsVolumeChartProps> = ({
  buckets,
  step,
  onZoom,
  onBack,
}) => {
  const { palette } = useTheme();
  const [drag, setDrag] = useState<[number, number] | null>(null);
  const stepMs = parseFloat(step) * 1000;
  const total = (b: VolumeBucket) => b.error + b.warning + b.info + b.other;
  const max = roundUp(Math.max(0, ...buckets.map(total)));
  const totals = CLASSES.map((c) => ({
    ...c,
    count: buckets.reduce((sum, b) => sum + b[c.key], 0),
  }));
  const first = buckets.length ? Date.parse(buckets[0].start) : 0;
  const last = buckets.length
    ? Date.parse(buckets[buckets.length - 1].start) + stepMs
    : 0;
  const tickFormat = last - first > DAY_MS ? formatTime : formatClock;
  const tickCount = Math.min(MAX_TIME_LABELS, buckets.length);
  const ticks = Array.from({ length: tickCount }, (_, k) =>
    Math.floor(((k + 0.5) * buckets.length) / tickCount)
  );

  useEffect(() => {
    if (!drag) return;
    const cancel = () => setDrag(null);
    window.addEventListener('mouseup', cancel);
    return () => window.removeEventListener('mouseup', cancel);
  }, [drag]);

  const finish = () => {
    if (!drag) return;
    const [a, b] = [Math.min(...drag), Math.max(...drag)];
    setDrag(null);
    onZoom(Date.parse(buckets[a].start), Date.parse(buckets[b].start) + stepMs);
  };

  const inDrag = (i: number) =>
    !!drag && i >= Math.min(...drag) && i <= Math.max(...drag);

  return (
    <Paper
      variant="outlined"
      data-testid="logs-volume-chart"
      sx={{ p: 2, pb: 1, borderRadius: 3, userSelect: 'none' }}
    >
      <Stack
        direction="row"
        alignItems="center"
        sx={{ mb: 1.5, gap: 2.5, flexWrap: 'wrap' }}
      >
        {totals.map((c) => (
          <Typography
            key={c.key}
            variant="body2"
            color="text.secondary"
            data-testid={`logs-volume-total-${c.key}`}
            sx={{ opacity: c.count ? 1 : 0.5 }}
          >
            <Box
              component="span"
              sx={{
                display: 'inline-block',
                width: 8,
                height: 8,
                mr: 0.75,
                borderRadius: '2px',
                bgcolor: severityColor(c.key, palette.mode),
              }}
            />
            {c.label}{' '}
            <Box
              component="span"
              sx={{
                color: 'text.primary',
                fontWeight: 600,
                fontVariantNumeric: 'tabular-nums',
              }}
            >
              {c.count.toLocaleString()}
            </Box>
          </Typography>
        ))}
        {onBack && (
          <Button
            size="small"
            startIcon={<ArrowBackIcon />}
            onClick={onBack}
            sx={{ ml: 'auto', textTransform: 'none' }}
          >
            Back to previous range
          </Button>
        )}
        <Typography
          variant="caption"
          color="text.secondary"
          sx={{ ml: onBack ? 0 : 'auto' }}
        >
          Drag across bars to zoom · {formatStep(stepMs)} per bar
        </Typography>
      </Stack>
      <Box sx={{ position: 'relative', height: HEIGHT, ml: `${AXIS_WIDTH}px` }}>
        {[max, max / 2, 0].map((value, i) => (
          <Box
            key={value}
            sx={{
              position: 'absolute',
              left: 0,
              right: 0,
              top: `${i * 50}%`,
              borderTop: '1px dashed',
              borderColor: 'divider',
            }}
          >
            <Typography
              variant="caption"
              color="text.secondary"
              data-testid="logs-volume-count-label"
              sx={{
                position: 'absolute',
                right: '100%',
                mr: 1,
                transform: 'translateY(-50%)',
                fontVariantNumeric: 'tabular-nums',
              }}
            >
              {compact.format(value)}
            </Typography>
          </Box>
        ))}
        <Stack
          direction="row"
          alignItems="flex-end"
          sx={{ position: 'absolute', inset: 0, gap: '2px' }}
        >
          {buckets.map((b, i) => (
            <Tooltip
              key={b.start}
              placement="top"
              title={
                <>
                  <div>{formatTime(Date.parse(b.start))}</div>
                  {CLASSES.map((c) => (
                    <div key={c.key}>
                      {c.label}: {b[c.key].toLocaleString()}
                    </div>
                  ))}
                </>
              }
            >
              <Stack
                data-testid="logs-volume-bucket"
                direction="column-reverse"
                onMouseDown={() => setDrag([i, i])}
                onMouseEnter={() => drag && setDrag([drag[0], i])}
                onMouseUp={finish}
                sx={{
                  flex: 1,
                  minWidth: 0,
                  height: '100%',
                  cursor: 'crosshair',
                  bgcolor: inDrag(i) ? 'action.selected' : 'transparent',
                }}
              >
                {CLASSES.map((c) => (
                  <Box
                    key={c.key}
                    sx={{
                      height: `${(b[c.key] / max) * 100}%`,
                      bgcolor: severityColor(c.key, palette.mode),
                    }}
                  />
                ))}
              </Stack>
            </Tooltip>
          ))}
        </Stack>
      </Box>
      <Box sx={{ position: 'relative', height: 22, ml: `${AXIS_WIDTH}px` }}>
        {ticks.map((i) => (
          <Typography
            key={i}
            variant="caption"
            color="text.secondary"
            data-testid="logs-volume-time-label"
            sx={{
              position: 'absolute',
              top: 4,
              left: `${((i + 0.5) / buckets.length) * 100}%`,
              transform: 'translateX(-50%)',
              whiteSpace: 'nowrap',
              fontVariantNumeric: 'tabular-nums',
            }}
          >
            {tickFormat(Date.parse(buckets[i].start))}
          </Typography>
        ))}
      </Box>
    </Paper>
  );
};

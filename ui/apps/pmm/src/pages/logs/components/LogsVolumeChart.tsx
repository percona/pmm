import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import {
  alpha,
  Box,
  Button,
  Chip,
  IconButton,
  Paper,
  Stack,
  Tooltip,
  Typography,
  useTheme,
} from '@mui/material';
import { FC, useEffect, useState } from 'react';
import { SeverityClass, VolumeBucket } from 'api/logs';
import { MONO, severityColor } from '../logsTheme';

/** One pattern drawn over the bars, with the time of the line it came from. */
export interface ChartOverlay {
  /** Lines of the pattern per bucket, keyed by the bucket's start. */
  counts: Map<string, number>;
  label: string;
  markerMs?: number;
  onClear: () => void;
}

interface LogsVolumeChartProps {
  buckets: VolumeBucket[];
  /** Bucket width as a protobuf Duration, e.g. "60s". */
  step: string;
  onZoom: (fromMs: number, toMs: number) => void;
  /** Returns to the range before the last zoom; the button shows only when set. */
  onBack?: () => void;
  /** The severity class the lines are filtered to, highlighted in the legend. */
  selectedClass?: SeverityClass;
  /** Filters to a class clicked in the legend; legend entries are buttons only when set. */
  onSelectClass?: (cls: SeverityClass) => void;
  overlay?: ChartOverlay;
}

const CLASSES: { key: SeverityClass; label: string }[] = [
  { key: 'error', label: 'Error' },
  { key: 'warning', label: 'Warning' },
  { key: 'info', label: 'Info' },
  { key: 'other', label: 'Other' },
];

const HEIGHT = 80;
const COLLAPSED_KEY = 'pmm-logs-chart-collapsed';

// A per-viewer preference; storage may be unavailable.
const readCollapsed = () => {
  try {
    return localStorage.getItem(COLLAPSED_KEY) === '1';
  } catch {
    return false;
  }
};

const writeCollapsed = (collapsed: boolean) => {
  try {
    localStorage.setItem(COLLAPSED_KEY, collapsed ? '1' : '0');
  } catch {
    // Not remembered.
  }
};
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
  selectedClass,
  onSelectClass,
  overlay,
}) => {
  const { palette } = useTheme();
  const [drag, setDrag] = useState<[number, number] | null>(null);
  const [collapsed, setCollapsed] = useState(readCollapsed);
  const toggleCollapsed = () => {
    writeCollapsed(!collapsed);
    setCollapsed(!collapsed);
  };
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

  const overlayCount = (b: VolumeBucket) => overlay?.counts.get(b.start) ?? 0;
  // The pattern has its own scale: a few lines a minute would vanish under busy bars.
  const overlayMax = roundUp(Math.max(0, ...buckets.map(overlayCount)));
  const overlayPoints = buckets
    .map((b, i) => `${i + 0.5},${overlayMax - overlayCount(b)}`)
    .join(' ');
  const plotSx = {
    position: 'relative',
    ml: `${AXIS_WIDTH}px`,
    mr: overlay ? `${AXIS_WIDTH}px` : 0,
  } as const;
  const markerPct =
    overlay?.markerMs !== undefined &&
    overlay.markerMs >= first &&
    overlay.markerMs <= last &&
    last > first
      ? ((overlay.markerMs - first) / (last - first)) * 100
      : undefined;

  return (
    <Paper
      variant="outlined"
      data-testid="logs-volume-chart"
      sx={{ px: 2, py: 1, borderRadius: 3, userSelect: 'none', flexShrink: 0 }}
    >
      <Stack
        direction="row"
        alignItems="center"
        sx={{
          mb: collapsed ? 0 : 1,
          gap: 2.5,
          flexWrap: 'nowrap',
          minWidth: 0,
        }}
      >
        {totals.map((c) => (
          <Typography
            key={c.key}
            variant="body2"
            color="text.secondary"
            data-testid={`logs-volume-total-${c.key}`}
            {...(onSelectClass && {
              component: 'button',
              type: 'button',
              'aria-pressed': selectedClass === c.key,
              title: `Show only ${c.label.toLowerCase()} lines`,
              onClick: () => onSelectClass(c.key),
            })}
            sx={{
              flexShrink: 0,
              whiteSpace: 'nowrap',
              px: 1,
              py: 0.25,
              ml: -1,
              border: 1,
              borderRadius: 2,
              font: 'inherit',
              fontSize: 'body2.fontSize',
              borderColor:
                selectedClass === c.key
                  ? severityColor(c.key, palette.mode)
                  : 'transparent',
              bgcolor:
                selectedClass === c.key
                  ? alpha(severityColor(c.key, palette.mode), 0.12)
                  : 'transparent',
              opacity: c.count || selectedClass === c.key ? 1 : 0.5,
              cursor: onSelectClass ? 'pointer' : 'default',
              '&:hover': onSelectClass ? { bgcolor: 'action.hover' } : {},
            }}
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
        {overlay && (
          <Chip
            size="small"
            label={`Showing: ${overlay.label}`}
            onDelete={overlay.onClear}
            title={overlay.label}
            data-testid="logs-volume-overlay-chip"
            sx={{
              minWidth: 0,
              flexShrink: 1,
              maxWidth: 420,
              ...MONO,
              fontSize: 12,
              color: palette.primary.main,
              borderColor: palette.primary.main,
            }}
            variant="outlined"
          />
        )}
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
          sx={{ ml: onBack ? 0 : 'auto', flexShrink: 0, whiteSpace: 'nowrap' }}
        >
          Drag across bars to zoom · {formatStep(stepMs)} per bar
        </Typography>
        <IconButton
          size="small"
          aria-label={collapsed ? 'Show chart' : 'Hide chart'}
          onClick={toggleCollapsed}
          sx={{ ml: -1.5 }}
        >
          {collapsed ? <ExpandMoreIcon /> : <ExpandLessIcon />}
        </IconButton>
      </Stack>
      {!collapsed && (
        <>
          <Box sx={{ ...plotSx, height: HEIGHT }}>
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
                {overlay && (
                  <Typography
                    variant="caption"
                    data-testid={
                      i === 0 ? 'logs-volume-overlay-max' : undefined
                    }
                    sx={{
                      position: 'absolute',
                      left: '100%',
                      ml: 1,
                      transform: 'translateY(-50%)',
                      fontVariantNumeric: 'tabular-nums',
                      color: palette.primary.main,
                    }}
                  >
                    {compact.format((overlayMax * (2 - i)) / 2)}
                  </Typography>
                )}
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
                      {overlay && (
                        <div>
                          This pattern: {overlayCount(b).toLocaleString()}
                        </div>
                      )}
                    </>
                  }
                >
                  <Stack
                    data-testid="logs-volume-bucket"
                    direction="column-reverse"
                    onMouseDown={() => setDrag([i, i])}
                    onMouseEnter={() => drag && setDrag([drag[0], i])}
                    onMouseUp={finish}
                    style={overlay ? { opacity: 0.35 } : undefined}
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
            {overlay && (
              <Box
                component="svg"
                data-testid="logs-volume-overlay"
                viewBox={`0 0 ${buckets.length} ${overlayMax}`}
                preserveAspectRatio="none"
                sx={{
                  position: 'absolute',
                  inset: 0,
                  width: '100%',
                  height: '100%',
                  pointerEvents: 'none',
                  overflow: 'visible',
                }}
              >
                <polygon
                  points={`0.5,${overlayMax} ${overlayPoints} ${buckets.length - 0.5},${overlayMax}`}
                  fill={alpha(palette.primary.main, 0.25)}
                />
                <polyline
                  points={overlayPoints}
                  fill="none"
                  stroke={palette.primary.main}
                  strokeWidth={2}
                  vectorEffect="non-scaling-stroke"
                />
              </Box>
            )}
            {markerPct !== undefined && (
              <Box
                data-testid="logs-volume-marker"
                title="The opened line"
                style={{ left: `${markerPct}%` }}
                sx={{
                  position: 'absolute',
                  top: -4,
                  bottom: 0,
                  borderLeft: '2px dashed',
                  borderColor: palette.primary.main,
                  pointerEvents: 'none',
                }}
              />
            )}
          </Box>
          <Box sx={{ ...plotSx, height: 22 }}>
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
        </>
      )}
    </Paper>
  );
};

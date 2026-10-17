import { Box, Stack, Tooltip, Typography } from '@mui/material';
import { FC, useEffect, useState } from 'react';
import { SeverityClass, VolumeBucket } from 'api/logs';

interface LogsVolumeChartProps {
  buckets: VolumeBucket[];
  /** Bucket width as a protobuf Duration, e.g. "60s". */
  step: string;
  onZoom: (fromMs: number, toMs: number) => void;
}

const CLASSES: { key: SeverityClass; color: string; label: string }[] = [
  { key: 'error', color: 'error.main', label: 'Error' },
  { key: 'warning', color: 'warning.main', label: 'Warning' },
  { key: 'info', color: 'info.main', label: 'Info' },
  { key: 'other', color: 'action.disabled', label: 'Other' },
];

const HEIGHT = 96;

const formatTime = (ms: number) =>
  new Date(ms).toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });

/** Stacked log counts per bucket; dragging across bars zooms to them. */
export const LogsVolumeChart: FC<LogsVolumeChartProps> = ({
  buckets,
  step,
  onZoom,
}) => {
  const [drag, setDrag] = useState<[number, number] | null>(null);
  const stepMs = parseFloat(step) * 1000;
  const total = (b: VolumeBucket) => b.error + b.warning + b.info + b.other;
  const max = Math.max(1, ...buckets.map(total));

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
    <Box data-testid="logs-volume-chart" sx={{ userSelect: 'none' }}>
      <Stack
        direction="row"
        alignItems="flex-end"
        sx={{ height: HEIGHT, gap: '1px' }}
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
                minWidth: 1,
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
                    bgcolor: c.color,
                  }}
                />
              ))}
            </Stack>
          </Tooltip>
        ))}
      </Stack>
      {buckets.length > 0 && (
        <Stack direction="row" justifyContent="space-between">
          <Typography variant="caption" color="text.secondary">
            {formatTime(Date.parse(buckets[0].start))}
          </Typography>
          <Typography variant="caption" color="text.secondary">
            {formatTime(Date.parse(buckets[buckets.length - 1].start) + stepMs)}
          </Typography>
        </Stack>
      )}
    </Box>
  );
};

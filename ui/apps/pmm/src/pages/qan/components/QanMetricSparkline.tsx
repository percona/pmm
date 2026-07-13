import { Box, Stack } from '@mui/material';
import { FC, useMemo } from 'react';
import type { QanMetricPoint } from 'types/qan.types';
import { bucketSparklineValues } from '../utils/qanDisplay';

interface QanMetricSparklineProps {
  points?: QanMetricPoint[];
  width?: number;
}

export const QanMetricSparkline: FC<QanMetricSparklineProps> = ({
  points,
  width = 96,
}) => {
  const segments = useMemo(() => bucketSparklineValues(points), [points]);

  if (!segments.length) {
    return <Box sx={{ width, height: 4 }} data-testid="qan-sparkline-empty" />;
  }

  const sorted = [...segments].sort((a, b) => a - b);
  const threshold = sorted[Math.floor(sorted.length / 2)] ?? 0;

  return (
    <Stack
      direction="row"
      spacing="2px"
      sx={{ width, height: 4, flexShrink: 0 }}
      data-testid="qan-sparkline"
    >
      {segments.map((value, idx) => (
        <Box
          key={idx}
          sx={{
            flex: 1,
            height: '100%',
            minWidth: 2,
            bgcolor: value >= threshold && value > 0 ? 'primary.light' : 'action.disabledBackground',
            borderRadius: 0.25,
          }}
        />
      ))}
    </Stack>
  );
};

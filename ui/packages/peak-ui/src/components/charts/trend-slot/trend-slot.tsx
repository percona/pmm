import Box from '@mui/material/Box';
import Skeleton from '@mui/material/Skeleton';
import Typography from '@mui/material/Typography';
import { SPARKLINE_DEFAULT_HEIGHT } from '../charts.constants';
import Sparkline from '../sparkline';
import type { TrendSlotProps } from './trend-slot.types';

const TrendSlot = ({
  sparkline,
  loading,
  emptyText,
  variant,
  defaultHeight = SPARKLINE_DEFAULT_HEIGHT,
}: TrendSlotProps) => {
  const height = sparkline?.height ?? defaultHeight;
  if (loading) return <Skeleton variant="rounded" height={height} />;
  if (!sparkline) return null;
  if (sparkline.data.length === 0) {
    return (
      <Box height={height} display="flex" alignItems="center">
        <Typography variant="caption">{emptyText}</Typography>
      </Box>
    );
  }
  return <Sparkline variant={variant} {...sparkline} height={height} />;
};

export default TrendSlot;

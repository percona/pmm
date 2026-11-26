import { Fragment, forwardRef } from 'react';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import visuallyHidden from '@mui/utils/visuallyHidden';
import { getChartColor } from '../../../design/chart-colors';
import { CHARTS_EMPTY_TEXT, TABULAR_NUMS } from '../charts.constants';
import { formatShare } from '../charts.utils';
import type { BarListProps } from './bar-list.types';

const BarList = forwardRef<HTMLDListElement, BarListProps>(
  (
    {
      items,
      valueFormatter = formatShare,
      color = 'primary',
      emptyLabel = '—',
      emptyText = CHARTS_EMPTY_TEXT,
      ...rest
    },
    ref
  ) => {
    const theme = useTheme();
    const max = Math.max(0, ...items.map((item) => item.value));

    return (
      <Box
        ref={ref}
        component="dl"
        m={0}
        display="grid"
        gridTemplateColumns="max-content 1fr"
        columnGap={1.5}
        rowGap={0.5}
        alignItems="center"
        {...rest}
      >
        {items.map(({ label, value }) => (
          <Fragment key={label}>
            <Typography
              component="dt"
              variant="caption"
              textAlign="right"
              sx={TABULAR_NUMS}
            >
              {label}
            </Typography>
            <Box
              component="dd"
              m={0}
              display="flex"
              alignItems="center"
              gap={1}
              minWidth={0}
            >
              {value > 0 ? (
                <>
                  <Box
                    aria-hidden
                    data-bar
                    sx={{
                      width: `${(value / max) * 80}%`,
                      minWidth: 3,
                      height: 16,
                      borderRadius: `${theme.shape.borderRadiusXs}px`,
                      bgcolor: getChartColor(theme, color),
                    }}
                  />
                  <Typography variant="caption" sx={TABULAR_NUMS}>
                    {valueFormatter(value)}
                  </Typography>
                </>
              ) : (
                <Typography variant="caption">
                  <span aria-hidden>{emptyLabel}</span>
                  <Box component="span" sx={visuallyHidden}>
                    {emptyText}
                  </Box>
                </Typography>
              )}
            </Box>
          </Fragment>
        ))}
      </Box>
    );
  }
);

BarList.displayName = 'BarList';

export default BarList;

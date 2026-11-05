import { Box, Typography } from '@mui/material';
import { FC } from 'react';

interface QanQueryCellProps {
  fingerprint?: string;
  dimension?: string;
  isTotals?: boolean;
}

/** Compact fingerprint preview matching Figma listing rows. */
export const QanQueryCell: FC<QanQueryCellProps> = ({
  fingerprint,
  dimension,
  isTotals,
}) => {
  if (isTotals) {
    return (
      <Typography variant="subtitle2" sx={{ fontWeight: 600, pl: 1 }}>
        Totals
      </Typography>
    );
  }

  const text = fingerprint || dimension || 'N/A';
  const singleLine = text.replace(/\s+/g, ' ').trim();

  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        minHeight: 40,
        px: 1.5,
        py: 0.5,
        borderRadius: 0.5,
        border: 1,
        borderColor: 'divider',
        bgcolor: (theme) =>
          theme.palette.mode === 'dark'
            ? 'rgba(32, 68, 147, 0.08)'
            : 'background.default',
        overflow: 'hidden',
      }}
      title={text}
    >
      <Typography
        component="span"
        sx={{
          fontFamily: '"Roboto Mono", monospace',
          fontSize: 14,
          fontWeight: 450,
          lineHeight: 1.375,
          whiteSpace: 'nowrap',
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          color: 'text.primary',
        }}
      >
        {singleLine}
      </Typography>
    </Box>
  );
};

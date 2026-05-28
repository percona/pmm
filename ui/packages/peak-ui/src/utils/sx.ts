import type { SxProps, Theme } from '@mui/material';

const flattenSx = (sx: SxProps<Theme>, theme: Theme): Record<string, unknown> => {
  if (typeof sx === 'function') return (sx as (t: Theme) => Record<string, unknown>)(theme);
  if (Array.isArray(sx))
    return Object.assign(
      {},
      ...sx.filter(Boolean).map((s) => flattenSx(s as SxProps<Theme>, theme))
    );
  return (sx ?? {}) as Record<string, unknown>;
};

const mergeSx = (
  defaultSx: SxProps<Theme> | undefined,
  consumerSx: SxProps<Theme> | undefined
): SxProps<Theme> | undefined => {
  if (!consumerSx) return defaultSx;
  if (!defaultSx) return consumerSx;
  return ((theme: Theme) => ({
    ...flattenSx(defaultSx, theme),
    ...flattenSx(consumerSx, theme),
  })) as SxProps<Theme>;
};

export { flattenSx, mergeSx };

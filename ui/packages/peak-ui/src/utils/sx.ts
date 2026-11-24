import type { SxProps, Theme } from '@mui/material';
import deepmerge from '@mui/utils/deepmerge';

type StyleObject = Record<string, unknown>;

const flattenSx = (sx: SxProps<Theme>, theme: Theme): StyleObject => {
  if (typeof sx === 'function') return (sx as (t: Theme) => StyleObject)(theme);
  if (Array.isArray(sx))
    return sx
      .filter(Boolean)
      .reduce<StyleObject>(
        (merged, s) => deepmerge(merged, flattenSx(s as SxProps<Theme>, theme)),
        {}
      );
  return (sx ?? {}) as StyleObject;
};

const mergeSx = (
  defaultSx: SxProps<Theme> | undefined,
  consumerSx: SxProps<Theme> | undefined
): SxProps<Theme> | undefined => {
  if (!consumerSx) return defaultSx;
  if (!defaultSx) return consumerSx;
  return ((theme: Theme) =>
    deepmerge(flattenSx(defaultSx, theme), flattenSx(consumerSx, theme))) as SxProps<Theme>;
};

export { flattenSx, mergeSx };

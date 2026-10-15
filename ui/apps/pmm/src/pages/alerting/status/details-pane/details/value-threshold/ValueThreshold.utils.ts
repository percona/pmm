export const clampFraction = (fraction: number): number =>
  Number.isFinite(fraction) ? Math.min(Math.max(fraction, 0), 1) : 0;

export const toPercent = (fraction: number): string =>
  `${clampFraction(fraction) * 100}%`;

// Round to at most 2 decimals and drop trailing zeros (2.6490066 -> "2.65", 80 -> "80").
export const formatNumber = (value: number): string =>
  `${Number(value.toFixed(2))}`;

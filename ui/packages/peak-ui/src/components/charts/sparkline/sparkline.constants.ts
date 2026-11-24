import SparklineTooltip from './sparkline-tooltip';

export const SPARKLINE_NUMBER_FORMAT = new Intl.NumberFormat(undefined, {
  maximumFractionDigits: 2,
});
export const SPARKLINE_MARGIN = { top: 2, right: 0, bottom: 0, left: 0 };
export const SPARKLINE_HIDDEN = { 'aria-hidden': true };
export const SPARKLINE_SLOTS = { tooltip: SparklineTooltip };

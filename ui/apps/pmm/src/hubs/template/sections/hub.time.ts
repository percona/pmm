// The format module alone: the plugin's barrel also loads its Automations pages, which this app's tests cannot import.
import { formatAge, formatCompactDuration } from '@pmm/plugins-om/src/format';

export { formatAge };

/** How long ago an ISO time was, as "18m", or null without a usable time. */
export const formatSince = (iso?: string, now: number = Date.now()) => {
  const at = iso ? Date.parse(iso) : Number.NaN;

  return Number.isNaN(at)
    ? null
    : formatCompactDuration(Math.max(0, (now - at) / 1000)) || '0s';
};

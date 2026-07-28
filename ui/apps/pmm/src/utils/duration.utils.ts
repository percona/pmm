export const DURATION_REGEX = /(-?\d+(\.\d+)?)(ns|us|µs|ms|s|m|h)/g;

/**
 * Format a duration given in seconds as a compact Prometheus-style string,
 * e.g. 30 -> "30s", 300 -> "5m", 5400 -> "1h 30m". Sub-second values are
 * rendered in milliseconds (e.g. 0.015 -> "15ms"). Returns an empty string for
 * missing values so callers can fall back to their own "unavailable" rendering.
 */
export const formatDurationSeconds = (seconds?: number): string => {
  if (seconds === undefined || seconds === null || Number.isNaN(seconds)) {
    return '';
  }

  if (seconds === 0) {
    return '0s';
  }

  if (seconds < 1) {
    return `${Math.round(seconds * 1000)}ms`;
  }

  const totalSeconds = Math.floor(seconds);
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const secs = totalSeconds % 60;

  const parts: string[] = [];
  if (hours > 0) {
    parts.push(`${hours}h`);
  }
  if (minutes > 0) {
    parts.push(`${minutes}m`);
  }
  if (secs > 0) {
    parts.push(`${secs}s`);
  }

  return parts.join(' ');
};

export const parseDuration = (duration: string): number => {
  let totalMs = 0;
  let match;
  while ((match = DURATION_REGEX.exec(duration)) !== null) {
    const value = parseFloat(match[1]);
    switch (match[3]) {
      case 'ns':
        totalMs += value / 1e6;
        break;
      case 'us':
      case 'µs':
        totalMs += value / 1e3;
        break;
      case 'ms':
        totalMs += value;
        break;
      case 's':
        totalMs += value * 1000;
        break;
      case 'm':
        totalMs += value * 60000;
        break;
      case 'h':
        totalMs += value * 3600000;
        break;
    }
  }
  return totalMs;
};

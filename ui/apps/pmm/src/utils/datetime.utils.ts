/**
 * Calculate the difference in milliseconds between a timestamp and now.
 */
export const diffFromNow = (timestamp: string): number => {
  const timestampDate = new Date(timestamp);
  const now = new Date();

  return now.getTime() - timestampDate.getTime();
};

export const formatTimestamp = (timestamp: string) =>
  new Date(timestamp).toLocaleDateString('en-US', {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  });

export const formatDuration = (duration: number) => {
  const hours = Math.floor(duration / 3600000);
  const minutes = Math.floor((duration % 3600000) / 60000);

  if (hours > 0) {
    return `${hours}h ${minutes}m`;
  }

  return `${minutes} min`;
};

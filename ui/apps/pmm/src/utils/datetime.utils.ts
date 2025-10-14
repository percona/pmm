/**
 * Calculate the difference in milliseconds between a timestamp and now.
 */
export const diffFromNow = (timestamp: string): number => {
  const timestampDate = new Date(timestamp);
  const now = new Date();

  return timestampDate.getTime() - now.getTime();
};

export const formatTimestamp = (timestamp: string) =>
  new Date(timestamp).toLocaleDateString('en-US', {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  });

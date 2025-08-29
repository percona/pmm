export const formatTimestamp = (timestamp: string) =>
  new Date(timestamp).toLocaleDateString('en-US', {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  });

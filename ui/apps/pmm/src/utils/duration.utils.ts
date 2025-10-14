export const parseDuration = (duration: string): number => {
  const regex = /(-?\d+(\.\d+)?)(ns|us|µs|ms|s|m|h)/g;
  let totalMs = 0;
  let match;
  while ((match = regex.exec(duration)) !== null) {
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

import { GetHANodeResponse, HAHealth } from 'types/ha.types';

export const getHAHealth = (nodes: GetHANodeResponse[]): HAHealth => {
  const nonAliveCount = nodes.filter((node) => node.status !== 'alive').length;

  if (nonAliveCount === nodes.length) {
    return 'down';
  }

  if (nonAliveCount >= 2 * (nodes.length / 3.0)) {
    return 'critical';
  }

  if (nonAliveCount >= nodes.length / 3.0) {
    return 'degraded';
  }

  return 'healthy';
};

import { GetHANodeResponse, HAHealth } from 'types/ha.types';

export const getHAHealth = (
  nodes: GetHANodeResponse[],
  expectedNodes: number
): HAHealth => {
  const nonAliveCount =
    expectedNodes - nodes.filter((node) => node.status === 'alive').length;

  if (nonAliveCount === expectedNodes) {
    return 'down';
  }

  if (nonAliveCount >= 2 * (expectedNodes / 3.0)) {
    return 'critical';
  }

  if (nonAliveCount >= expectedNodes / 3.0) {
    return 'degraded';
  }

  return 'healthy';
};

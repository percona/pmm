import { UseQueryResult } from '@tanstack/react-query';
import {
  GetHANodeResponse,
  GetHANodesResponse,
  HAHealth,
} from 'types/ha.types';

export const getHAHealth = (
  nodes?: GetHANodeResponse[],
  expectedNodes?: number,
  query?: UseQueryResult<GetHANodesResponse, Error>
): HAHealth => {
  if (query?.isLoading || nodes === undefined || expectedNodes === undefined) {
    return 'unknown';
  }

  const nonAliveCount =
    expectedNodes - nodes.filter((node) => node.status === 'alive').length;

  if (
    nonAliveCount === expectedNodes ||
    query?.isError ||
    query?.fetchStatus === 'paused'
  ) {
    return 'unreachable';
  }

  if (nonAliveCount >= 2 * (expectedNodes / 3.0)) {
    return 'critical';
  }

  if (nonAliveCount >= expectedNodes / 3.0) {
    return 'degraded';
  }

  return 'healthy';
};

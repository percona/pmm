import { useQuery, UseQueryOptions } from '@tanstack/react-query';
import { getHANodes, getHAStatus } from 'api/ha';
import {
  GetHANodesResponse,
  GetHAStatusResponse,
  NodeRole,
} from 'types/ha.types';
import { getHAHealth } from 'utils/ha.utils';

export const useHAStatus = (
  options?: Partial<UseQueryOptions<GetHAStatusResponse>>
) =>
  useQuery({
    queryKey: ['ha:status'],
    queryFn: () => getHAStatus(),
    ...options,
  });

export const useHANodes = (
  options?: Partial<UseQueryOptions<GetHANodesResponse>>
) =>
  useQuery({
    queryKey: ['ha:nodes'],
    queryFn: () => getHANodes(),
    ...options,
  });

export const useHaInfo = (
  options?: Partial<UseQueryOptions<GetHAStatusResponse>>
) => {
  const statusQuery = useHAStatus(options);
  const nodesQuery = useHANodes({
    enabled: statusQuery.data?.status === 'Enabled',
    refetchInterval: 15000,
  });

  const health = getHAHealth(
    nodesQuery.data?.nodes || [],
    nodesQuery.data?.expectedNodes || 0,
    nodesQuery
  );
  const enabled = statusQuery.data?.status === 'Enabled';
  const leader = nodesQuery.data?.nodes.find(
    (node) => node.role === NodeRole.leader
  );

  return {
    data: {
      enabled,
      health,
      leader,
      nodes: nodesQuery.data?.nodes || [],
    },
    isLoading: nodesQuery.isLoading || statusQuery.isLoading,
  };
};

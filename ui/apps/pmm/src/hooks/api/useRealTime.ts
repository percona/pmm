import {
  useMutation,
  UseMutationOptions,
  useQuery,
  UseQueryOptions,
} from '@tanstack/react-query';
import { changeRealTimeAgent, getRunningRealTimeAgents } from 'api/rta';
import {
  ChangeRealTimeAgentPayload,
  ChangeRealTimeAgentResponse,
  RunningRealTimeAgent,
} from 'types/rta.types';

export const useRealTimeAgents = (
  options?: Partial<UseQueryOptions<RunningRealTimeAgent[]>>
) =>
  useQuery({
    queryKey: ['rta:list-agents'],
    queryFn: () => getRunningRealTimeAgents(),
    ...options,
  });

export const useChangeRealTimeAgent = (
  options?: Partial<
    UseMutationOptions<
      ChangeRealTimeAgentResponse,
      Error,
      ChangeRealTimeAgentPayload
    >
  >
) =>
  useMutation({
    mutationKey: ['rta:change-agent'],
    mutationFn: (payload: ChangeRealTimeAgentPayload) =>
      changeRealTimeAgent(payload),
    ...options,
  });

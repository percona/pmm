import {
  keepPreviousData,
  useMutation,
  UseMutationOptions,
  useQuery,
  useQueryClient,
  UseQueryOptions,
} from '@tanstack/react-query';
import {
  changeAdvisorChecks,
  listAdvisors,
  listCheckResultsFilterValues,
  listCheckResultsHistory,
  markCheckResultsRead,
  startAdvisorChecks,
} from 'api/advisors';
import {
  Advisor,
  ChangeAdvisorCheckParams,
  CheckResultHistoryItem,
  ListCheckResultsFilterValuesResponse,
  ListCheckResultsHistoryParams,
  MarkCheckResultsReadRequest,
} from 'types/advisors.types';
import { PaginatedResponse } from 'types/util.types';

const KEYS = {
  LIST: 'advisors:list',
  START_CHECKS: 'advisors:start-checks',
  CHANGE_CHECKS: 'advisors:change-checks',
  HISTORY: 'advisors:history',
  HISTORY_FILTER_VALUES: 'advisors:history-filter-values',
  MARK_READ: 'advisors:mark-read',
};

export const useAdvisors = (options?: Partial<UseQueryOptions<Advisor[]>>) =>
  useQuery({
    queryKey: [KEYS.LIST],
    queryFn: () => listAdvisors(),
    ...options,
  });

export const useStartAdvisorChecks = (
  options?: Partial<UseMutationOptions<string, Error, string[]>>
) =>
  useMutation({
    mutationKey: [KEYS.START_CHECKS],
    mutationFn: startAdvisorChecks,
    ...options,
  });

export const useCheckResultsHistory = (
  params: ListCheckResultsHistoryParams,
  options?: Partial<UseQueryOptions<PaginatedResponse<CheckResultHistoryItem>>>
) =>
  useQuery({
    queryKey: [KEYS.HISTORY, params],
    queryFn: () => listCheckResultsHistory(params),
    // keep showing the current page while the next one loads
    placeholderData: keepPreviousData,
    ...options,
  });

export const useCheckResultsFilterValues = (
  options?: Partial<UseQueryOptions<ListCheckResultsFilterValuesResponse>>
) =>
  useQuery({
    queryKey: [KEYS.HISTORY_FILTER_VALUES],
    queryFn: () => listCheckResultsFilterValues(),
    ...options,
  });

export const useMarkCheckResultsRead = (
  options?: Partial<
    UseMutationOptions<void, Error, MarkCheckResultsReadRequest>
  >
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: [KEYS.MARK_READ],
    mutationFn: markCheckResultsRead,
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await queryClient.invalidateQueries({ queryKey: [KEYS.HISTORY] });
    },
  });
};

export const useChangeAdvisorChecks = (
  options?: Partial<UseMutationOptions<void, Error, ChangeAdvisorCheckParams[]>>
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: [KEYS.CHANGE_CHECKS],
    mutationFn: changeAdvisorChecks,
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await queryClient.invalidateQueries({ queryKey: [KEYS.LIST] });
    },
  });
};

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
  createAdvisorCheck,
  deleteAdvisorCheck,
  getAdvisorCheck,
  getAdvisorCheckScript,
  listAdvisors,
  listCheckResultsFilterValues,
  listCheckResultsHistory,
  markCheckResultsRead,
  startAdvisorChecks,
  updateAdvisorCheck,
} from 'api/advisors';
import {
  Advisor,
  AdvisorCheck,
  AdvisorCheckInput,
  ChangeAdvisorCheckParams,
  CheckResultHistoryItem,
  ListCheckResultsFilterValuesResponse,
  ListCheckResultsHistoryParams,
  MarkCheckResultsReadRequest,
} from 'types/advisors.types';
import { PaginatedResponse } from 'types/util.types';

const KEYS = {
  LIST: 'advisors:list',
  CHECK: 'advisors:check',
  CHECK_SCRIPT: 'advisors:check-script',
  START_CHECKS: 'advisors:start-checks',
  CHANGE_CHECKS: 'advisors:change-checks',
  CREATE_CHECK: 'advisors:create-check',
  UPDATE_CHECK: 'advisors:update-check',
  DELETE_CHECK: 'advisors:delete-check',
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

export const useAdvisorCheckScript = (
  name?: string,
  options?: Partial<UseQueryOptions<string>>
) =>
  useQuery({
    queryKey: [KEYS.CHECK_SCRIPT, name],
    queryFn: () => getAdvisorCheckScript(name!),
    // only fetch once a check is selected (lazy, on overlay open)
    enabled: !!name,
    ...options,
  });

export const useAdvisorCheck = (
  name?: string,
  options?: Partial<UseQueryOptions<AdvisorCheck>>
) =>
  useQuery({
    queryKey: [KEYS.CHECK, name],
    queryFn: () => getAdvisorCheck(name!),
    // only fetch once a check is selected (lazy, on overlay open)
    enabled: !!name,
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

export const useCreateAdvisorCheck = (
  options?: Partial<UseMutationOptions<AdvisorCheck, Error, AdvisorCheckInput>>
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: [KEYS.CREATE_CHECK],
    mutationFn: createAdvisorCheck,
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await queryClient.invalidateQueries({ queryKey: [KEYS.LIST] });
    },
  });
};

export const useUpdateAdvisorCheck = (
  options?: Partial<
    UseMutationOptions<
      AdvisorCheck,
      Error,
      { name: string; check: AdvisorCheckInput }
    >
  >
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: [KEYS.UPDATE_CHECK],
    mutationFn: ({ name, check }) => updateAdvisorCheck(name, check),
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await queryClient.invalidateQueries({ queryKey: [KEYS.LIST] });
      await queryClient.invalidateQueries({
        queryKey: [KEYS.CHECK, variables.name],
      });
    },
  });
};

export const useDeleteAdvisorCheck = (
  options?: Partial<UseMutationOptions<void, Error, string>>
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: [KEYS.DELETE_CHECK],
    mutationFn: deleteAdvisorCheck,
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await queryClient.invalidateQueries({ queryKey: [KEYS.LIST] });
    },
  });
};

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
  listAdvisorCheckTestTargets,
  listAdvisors,
  listInsightsFilterValues,
  listInsights,
  markInsightsRead,
  startAdvisorChecks,
  testAdvisorCheck,
  updateAdvisorCheck,
} from 'api/advisors';
import {
  Advisor,
  AdvisorCheck,
  AdvisorCheckInput,
  AdvisorCheckTestTarget,
  AdvisorTechnology,
  ChangeAdvisorCheckParams,
  Insight,
  ListInsightsFilterValuesResponse,
  ListInsightsParams,
  MarkInsightsReadRequest,
  TestAdvisorCheckRequest,
  TestAdvisorCheckResponse,
} from 'types/advisors.types';
import { PaginatedResponse } from 'types/util.types';

const KEYS = {
  LIST: 'advisors:list',
  CHECK: 'advisors:check',
  TEST_TARGETS: 'advisors:test-targets',
  START_CHECKS: 'advisors:start-checks',
  CHANGE_CHECKS: 'advisors:change-checks',
  CREATE_CHECK: 'advisors:create-check',
  UPDATE_CHECK: 'advisors:update-check',
  DELETE_CHECK: 'advisors:delete-check',
  TEST_CHECK: 'advisors:test-check',
  INSIGHTS: 'advisors:insights',
  INSIGHTS_FILTER_VALUES: 'advisors:insights-filter-values',
  MARK_READ: 'advisors:mark-read',
};

export const useAdvisors = (options?: Partial<UseQueryOptions<Advisor[]>>) =>
  useQuery({
    queryKey: [KEYS.LIST],
    queryFn: () => listAdvisors(),
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

export const useAdvisorCheckTestTargets = (
  technology?: AdvisorTechnology,
  options?: Partial<UseQueryOptions<AdvisorCheckTestTarget[]>>
) =>
  useQuery({
    queryKey: [KEYS.TEST_TARGETS, technology],
    queryFn: () => listAdvisorCheckTestTargets(technology!),
    enabled: !!technology,
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

export const useInsights = (
  params: ListInsightsParams,
  options?: Partial<UseQueryOptions<PaginatedResponse<Insight>>>
) =>
  useQuery({
    queryKey: [KEYS.INSIGHTS, params],
    queryFn: () => listInsights(params),
    // keep showing the current page while the next one loads
    placeholderData: keepPreviousData,
    ...options,
  });

export const useInsightsFilterValues = (
  options?: Partial<UseQueryOptions<ListInsightsFilterValuesResponse>>
) =>
  useQuery({
    queryKey: [KEYS.INSIGHTS_FILTER_VALUES],
    queryFn: () => listInsightsFilterValues(),
    ...options,
  });

export const useMarkInsightsRead = (
  options?: Partial<UseMutationOptions<void, Error, MarkInsightsReadRequest>>
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: [KEYS.MARK_READ],
    mutationFn: markInsightsRead,
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await queryClient.invalidateQueries({ queryKey: [KEYS.INSIGHTS] });
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

// a dry-run mutates nothing on the server, so no queries are invalidated
export const useTestAdvisorCheck = (
  options?: Partial<
    UseMutationOptions<TestAdvisorCheckResponse, Error, TestAdvisorCheckRequest>
  >
) =>
  useMutation({
    mutationKey: [KEYS.TEST_CHECK],
    mutationFn: testAdvisorCheck,
    ...options,
  });

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

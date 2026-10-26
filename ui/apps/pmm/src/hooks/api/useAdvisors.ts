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
  createAdvisorSchedule,
  deleteAdvisorCheck,
  deleteAdvisorSchedule,
  getAdvisorCheck,
  getAdvisorNotificationSettings,
  listAdvisorChecks,
  listAdvisorCheckTestTargets,
  listAdvisorSchedules,
  listAdvisorServices,
  listInsightsFilterValues,
  listInsights,
  listRuns,
  runAdvisorSchedule,
  sendTestAdvisorNotification,
  startAdvisorChecks,
  testAdvisorCheck,
  updateAdvisorCheck,
  updateAdvisorSchedule,
} from 'api/advisors';
import {
  AdvisorCheck,
  AdvisorCheckResultStatus,
  AdvisorNotificationSettings,
  AdvisorRun,
  AdvisorSchedule,
  AdvisorScheduleInput,
  AdvisorTargetService,
  AdvisorCheckInput,
  AdvisorCheckTestTarget,
  AdvisorTechnology,
  ChangeAdvisorCheckParams,
  Insight,
  ListInsightsFilterValuesResponse,
  ListInsightsParams,
  ListRunsParams,
  SendTestAdvisorNotificationRequest,
  StartAdvisorChecksRequest,
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
  RUNS: 'advisors:runs',
  INSIGHTS_FILTER_VALUES: 'advisors:insights-filter-values',
  TEST_NOTIFICATION: 'advisors:test-notification',
  SCHEDULES: 'advisors:schedules',
  SERVICES: 'advisors:services',
  NOTIFICATION_SETTINGS: 'advisors:notification-settings',
  SAVE_SCHEDULE: 'advisors:save-schedule',
  DELETE_SCHEDULE: 'advisors:delete-schedule',
  RUN_SCHEDULE: 'advisors:run-schedule',
};

export const useAdvisorChecks = (
  options?: Partial<UseQueryOptions<AdvisorCheck[]>>
) =>
  useQuery({
    queryKey: [KEYS.LIST],
    queryFn: () => listAdvisorChecks(),
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
  options?: Partial<
    UseMutationOptions<string, Error, StartAdvisorChecksRequest>
  >
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

// fetches every insight of a run, optionally only those with a status, on
// demand: without a page size the API returns them all
export const useFetchRunInsights = () => {
  const queryClient = useQueryClient();

  return (runId: string, status?: AdvisorCheckResultStatus) => {
    const params: ListInsightsParams = { runId, status };
    return queryClient.fetchQuery({
      queryKey: [KEYS.INSIGHTS, params],
      queryFn: () => listInsights(params),
    });
  };
};

export const useRuns = (
  params: ListRunsParams,
  options?: Partial<UseQueryOptions<PaginatedResponse<AdvisorRun>>>
) =>
  useQuery({
    queryKey: [KEYS.RUNS, params],
    queryFn: () => listRuns(params),
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

// sending a test email changes nothing on the server, so no queries are invalidated
export const useSendTestAdvisorNotification = (
  options?: Partial<
    UseMutationOptions<void, Error, SendTestAdvisorNotificationRequest>
  >
) =>
  useMutation({
    mutationKey: [KEYS.TEST_NOTIFICATION],
    mutationFn: sendTestAdvisorNotification,
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

export const useAdvisorSchedules = (
  options?: Partial<UseQueryOptions<AdvisorSchedule[]>>
) =>
  useQuery({
    queryKey: [KEYS.SCHEDULES],
    queryFn: () => listAdvisorSchedules(),
    ...options,
  });

export const useAdvisorServices = (
  options?: Partial<UseQueryOptions<AdvisorTargetService[]>>
) =>
  useQuery({
    queryKey: [KEYS.SERVICES],
    queryFn: () => listAdvisorServices(),
    ...options,
  });

export const useAdvisorNotificationSettings = (
  options?: Partial<UseQueryOptions<AdvisorNotificationSettings>>
) =>
  useQuery({
    queryKey: [KEYS.NOTIFICATION_SETTINGS],
    queryFn: () => getAdvisorNotificationSettings(),
    ...options,
  });

// creates a schedule, or replaces one when scheduleId is set
export const useSaveAdvisorSchedule = (
  options?: Partial<
    UseMutationOptions<
      AdvisorSchedule,
      Error,
      { scheduleId?: string; input: AdvisorScheduleInput }
    >
  >
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: [KEYS.SAVE_SCHEDULE],
    mutationFn: ({ scheduleId, input }) =>
      scheduleId
        ? updateAdvisorSchedule(scheduleId, input)
        : createAdvisorSchedule(input),
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await queryClient.invalidateQueries({ queryKey: [KEYS.SCHEDULES] });
    },
  });
};

export const useDeleteAdvisorSchedule = (
  options?: Partial<UseMutationOptions<void, Error, string>>
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: [KEYS.DELETE_SCHEDULE],
    mutationFn: deleteAdvisorSchedule,
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await queryClient.invalidateQueries({ queryKey: [KEYS.SCHEDULES] });
    },
  });
};

export const useRunAdvisorSchedule = (
  options?: Partial<UseMutationOptions<string, Error, string>>
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: [KEYS.RUN_SCHEDULE],
    mutationFn: runAdvisorSchedule,
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await queryClient.invalidateQueries({ queryKey: [KEYS.SCHEDULES] });
      await queryClient.invalidateQueries({ queryKey: [KEYS.RUNS] });
    },
  });
};

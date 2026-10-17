import type { AxiosRequestConfig, AxiosRequestHeaders } from 'axios';
import {
  Advisor,
  AdvisorCheck,
  AdvisorCheckInput,
  AdvisorCheckTestTarget,
  AdvisorRun,
  AdvisorTechnology,
  ChangeAdvisorCheckParams,
  ChangeAdvisorChecksRequest,
  Insight,
  CreateAdvisorCheckRequest,
  CreateAdvisorCheckResponse,
  GetAdvisorCheckResponse,
  ListAdvisorCheckTestTargetsResponse,
  ListAdvisorsResponse,
  ListInsightsFilterValuesResponse,
  ListInsightsParams,
  ListRunsParams,
  MarkInsightsReadRequest,
  SendTestAdvisorNotificationRequest,
  StartAdvisorChecksRequest,
  StartAdvisorChecksResponse,
  TestAdvisorCheckRequest,
  TestAdvisorCheckResponse,
  UpdateAdvisorCheckRequest,
  UpdateAdvisorCheckResponse,
} from 'types/advisors.types';
import { EmptyResponse, PaginatedResponse } from 'types/util.types';
import { api } from './api';

export const listAdvisors = async (): Promise<Advisor[]> => {
  const res = await api.get<ListAdvisorsResponse>('/advisors', rawKeysConfig);
  return res.data.advisors;
};

export const getAdvisorCheck = async (name: string): Promise<AdvisorCheck> => {
  const res = await api.get<GetAdvisorCheckResponse>(
    `/advisors/checks/${encodeURIComponent(name)}`,
    rawKeysConfig
  );
  return res.data.check;
};

export const createAdvisorCheck = async (
  check: AdvisorCheckInput
): Promise<AdvisorCheck> => {
  const payload: CreateAdvisorCheckRequest = { check };
  const res = await api.post<CreateAdvisorCheckResponse>(
    '/advisors/checks',
    payload,
    rawKeysConfig
  );
  return res.data.check;
};

export const updateAdvisorCheck = async (
  name: string,
  check: AdvisorCheckInput
): Promise<AdvisorCheck> => {
  const payload: UpdateAdvisorCheckRequest = { check };
  const res = await api.put<UpdateAdvisorCheckResponse>(
    `/advisors/checks/${encodeURIComponent(name)}`,
    payload,
    rawKeysConfig
  );
  return res.data.check;
};

export const deleteAdvisorCheck = async (name: string): Promise<void> => {
  await api.delete<EmptyResponse>(
    `/advisors/checks/${encodeURIComponent(name)}`
  );
};

export const startAdvisorChecks = async (
  payload: StartAdvisorChecksRequest
): Promise<string> => {
  const res = await api.post<StartAdvisorChecksResponse>(
    '/advisors/checks:start',
    payload
  );
  return res.data.runId;
};

export const testAdvisorCheck = async (
  payload: TestAdvisorCheckRequest
): Promise<TestAdvisorCheckResponse> => {
  // errors are rendered inside the form's test results panel,
  // so the global error snackbar is suppressed
  const res = await api.post<TestAdvisorCheckResponse>(
    '/advisors/checks:test',
    payload,
    { ...rawKeysConfig, disableNotifications: true }
  );
  return res.data;
};

export const listAdvisorCheckTestTargets = async (
  technology: AdvisorTechnology
): Promise<AdvisorCheckTestTarget[]> => {
  const res = await api.get<ListAdvisorCheckTestTargetsResponse>(
    '/advisors/checks:testTargets',
    { params: { technology } }
  );
  return res.data.targets ?? [];
};

// the failure reason (SMTP not configured, connection refused) is what the
// operator needs, and the interceptor already surfaces the server message
export const sendTestAdvisorNotification = async (
  payload: SendTestAdvisorNotificationRequest
): Promise<void> => {
  await api.post<EmptyResponse>('/advisors/notifications:test', payload);
};

export const changeAdvisorChecks = async (
  params: ChangeAdvisorCheckParams[]
): Promise<void> => {
  const payload: ChangeAdvisorChecksRequest = { params };
  await api.post<EmptyResponse>('/advisors/checks:batchChange', payload);
};

// Fields whose value is a free-form map keyed by data, not by a schema field
// name, so its keys must survive verbatim.
const RAW_KEY_FIELDS = ['labels', 'parameters'];

const camelizeKey = (key: string) =>
  key.replace(/_([a-z0-9])/g, (_, char: string) => char.toUpperCase());

const snakeizeKey = (key: string) =>
  key.replace(/[A-Z]/g, (char) => `_${char.toLowerCase()}`);

const transformKeys = (
  value: unknown,
  transformKey: (key: string) => string
): unknown => {
  if (Array.isArray(value)) {
    return value.map((item) => transformKeys(item, transformKey));
  }
  if (value === null || typeof value !== 'object') {
    return value;
  }
  return Object.fromEntries(
    Object.entries(value).map(([key, val]) => [
      transformKey(key),
      RAW_KEY_FIELDS.includes(key) ? val : transformKeys(val, transformKey),
    ])
  );
};

export const camelizeKeys = (value: unknown): unknown =>
  transformKeys(value, camelizeKey);

const snakeizeKeys = (value: unknown): unknown =>
  transformKeys(value, snakeizeKey);

// axios-case-converter rewrites every key recursively in both directions, which
// renames data-keyed maps (service_name -> serviceName, all_dbs -> allDbs).
// Endpoints carrying such maps replace the instance chains with these, so the
// JSON (de)serialization happens here too; falling back to the raw body on
// unparseable input keeps the error interceptor, which reads `data.message`,
// working as it does for every other endpoint.
const rawKeysConfig: AxiosRequestConfig = {
  transformRequest: [
    (data: unknown, headers: AxiosRequestHeaders) => {
      if (data === undefined) {
        return data;
      }
      headers.setContentType('application/json');
      return JSON.stringify(snakeizeKeys(data));
    },
  ],
  transformResponse: [
    (raw: string) => {
      if (typeof raw !== 'string' || !raw) {
        return raw;
      }
      try {
        return camelizeKeys(JSON.parse(raw));
      } catch {
        return raw;
      }
    },
  ],
};

export const listInsights = async (
  params: ListInsightsParams
): Promise<PaginatedResponse<Insight>> => {
  const res = await api.get<PaginatedResponse<Insight>>('/advisors/insights', {
    ...rawKeysConfig,
    params,
  });
  return res.data;
};

export const listRuns = async (
  params: ListRunsParams
): Promise<PaginatedResponse<AdvisorRun>> => {
  const res = await api.get<PaginatedResponse<AdvisorRun>>('/advisors/runs', {
    params,
  });
  return res.data;
};

export const markInsightsRead = async (
  payload: MarkInsightsReadRequest
): Promise<void> => {
  await api.post<EmptyResponse>('/advisors/insights:markRead', payload);
};

export const listInsightsFilterValues =
  async (): Promise<ListInsightsFilterValuesResponse> => {
    const res = await api.get<ListInsightsFilterValuesResponse>(
      '/advisors/insights:filterValues'
    );
    return res.data;
  };

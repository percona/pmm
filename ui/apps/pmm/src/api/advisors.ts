import {
  Advisor,
  AdvisorCheck,
  AdvisorCheckInput,
  AdvisorCheckTestTarget,
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
  MarkInsightsReadRequest,
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
  const res = await api.get<ListAdvisorsResponse>('/advisors');
  return res.data.advisors;
};

export const getAdvisorCheck = async (name: string): Promise<AdvisorCheck> => {
  const res = await api.get<GetAdvisorCheckResponse>(
    `/advisors/checks/${encodeURIComponent(name)}`
  );
  return res.data.check;
};

export const createAdvisorCheck = async (
  check: AdvisorCheckInput
): Promise<AdvisorCheck> => {
  const payload: CreateAdvisorCheckRequest = { check };
  const res = await api.post<CreateAdvisorCheckResponse>(
    '/advisors/checks',
    payload
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
    payload
  );
  return res.data.check;
};

export const deleteAdvisorCheck = async (name: string): Promise<void> => {
  await api.delete<EmptyResponse>(
    `/advisors/checks/${encodeURIComponent(name)}`
  );
};

export const startAdvisorChecks = async (names: string[]): Promise<string> => {
  const payload: StartAdvisorChecksRequest = { names };
  const res = await api.post<StartAdvisorChecksResponse>(
    '/advisors/checks:start',
    payload
  );
  return res.data.batchId;
};

export const testAdvisorCheck = async (
  payload: TestAdvisorCheckRequest
): Promise<TestAdvisorCheckResponse> => {
  // errors are rendered inside the form's test results panel,
  // so the global error snackbar is suppressed
  const res = await api.post<TestAdvisorCheckResponse>(
    '/advisors/checks:test',
    payload,
    { disableNotifications: true }
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

export const changeAdvisorChecks = async (
  params: ChangeAdvisorCheckParams[]
): Promise<void> => {
  const payload: ChangeAdvisorChecksRequest = { params };
  await api.post<EmptyResponse>('/advisors/checks:batchChange', payload);
};

export const listInsights = async (
  params: ListInsightsParams
): Promise<PaginatedResponse<Insight>> => {
  const res = await api.get<PaginatedResponse<Insight>>('/advisors/insights', {
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

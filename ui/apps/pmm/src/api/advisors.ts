import {
  Advisor,
  AdvisorCheck,
  AdvisorCheckInput,
  ChangeAdvisorCheckParams,
  ChangeAdvisorChecksRequest,
  CheckResultHistoryItem,
  CreateAdvisorCheckRequest,
  CreateAdvisorCheckResponse,
  GetAdvisorCheckResponse,
  GetAdvisorCheckScriptResponse,
  ListAdvisorsResponse,
  ListCheckResultsFilterValuesResponse,
  ListCheckResultsHistoryParams,
  MarkCheckResultsReadRequest,
  StartAdvisorChecksRequest,
  StartAdvisorChecksResponse,
  UpdateAdvisorCheckRequest,
  UpdateAdvisorCheckResponse,
} from 'types/advisors.types';
import { EmptyResponse, PaginatedResponse } from 'types/util.types';
import { api } from './api';

export const listAdvisors = async (): Promise<Advisor[]> => {
  const res = await api.get<ListAdvisorsResponse>('/advisors');
  return res.data.advisors;
};

export const getAdvisorCheckScript = async (name: string): Promise<string> => {
  const res = await api.get<GetAdvisorCheckScriptResponse>(
    `/advisors/checks/${encodeURIComponent(name)}/script`
  );
  return res.data.script;
};

export const getAdvisorCheck = async (name: string): Promise<AdvisorCheck> => {
  const res = await api.get<GetAdvisorCheckResponse>(
    `/advisors/checks/${encodeURIComponent(name)}/definition`
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

export const changeAdvisorChecks = async (
  params: ChangeAdvisorCheckParams[]
): Promise<void> => {
  const payload: ChangeAdvisorChecksRequest = { params };
  await api.post<EmptyResponse>('/advisors/checks:batchChange', payload);
};

export const listCheckResultsHistory = async (
  params: ListCheckResultsHistoryParams
): Promise<PaginatedResponse<CheckResultHistoryItem>> => {
  const res = await api.get<PaginatedResponse<CheckResultHistoryItem>>(
    '/advisors/checks/history',
    { params }
  );
  return res.data;
};

export const markCheckResultsRead = async (
  payload: MarkCheckResultsReadRequest
): Promise<void> => {
  await api.post<EmptyResponse>('/advisors/checks/history:markRead', payload);
};

export const listCheckResultsFilterValues =
  async (): Promise<ListCheckResultsFilterValuesResponse> => {
    const res = await api.get<ListCheckResultsFilterValuesResponse>(
      '/advisors/checks/history:filterValues'
    );
    return res.data;
  };

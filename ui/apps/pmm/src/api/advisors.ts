import {
  Advisor,
  ChangeAdvisorCheckParams,
  ChangeAdvisorChecksRequest,
  CheckResultHistoryItem,
  ListAdvisorsResponse,
  ListCheckResultsHistoryParams,
  MarkCheckResultsReadRequest,
  StartAdvisorChecksRequest,
  StartAdvisorChecksResponse,
} from 'types/advisors.types';
import { EmptyResponse, PaginatedResponse } from 'types/util.types';
import { api } from './api';

export const listAdvisors = async (): Promise<Advisor[]> => {
  const res = await api.get<ListAdvisorsResponse>('/advisors');
  return res.data.advisors;
};

export const startAdvisorChecks = async (
  names: string[]
): Promise<string> => {
  const payload: StartAdvisorChecksRequest = { names };
  const res = await api.post<StartAdvisorChecksResponse>(
    '/advisors/checks:start',
    payload
  );
  return res.data.runId;
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

import {
  Advisor,
  ChangeAdvisorCheckParams,
  ChangeAdvisorChecksRequest,
  CheckResultHistoryItem,
  ListAdvisorsResponse,
  ListCheckResultsFilterValuesResponse,
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

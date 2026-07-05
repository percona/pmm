import {
  Advisor,
  ChangeAdvisorCheckParams,
  ChangeAdvisorChecksRequest,
  ListAdvisorsResponse,
  StartAdvisorChecksRequest,
} from 'types/advisors.types';
import { EmptyResponse } from 'types/util.types';
import { api } from './api';

export const listAdvisors = async (): Promise<Advisor[]> => {
  const res = await api.get<ListAdvisorsResponse>('/advisors');
  return res.data.advisors;
};

export const startAdvisorChecks = async (names: string[]): Promise<void> => {
  const payload: StartAdvisorChecksRequest = { names };
  await api.post<EmptyResponse>('/advisors/checks:start', payload);
};

export const changeAdvisorChecks = async (
  params: ChangeAdvisorCheckParams[]
): Promise<void> => {
  const payload: ChangeAdvisorChecksRequest = { params };
  await api.post<EmptyResponse>('/advisors/checks:batchChange', payload);
};

import { get, post } from './api';
import type {
  ActionStarted,
  GetActionResponse,
  StartNodeActionBody,
  StartServiceActionBody,
  StartServiceActionResponse,
} from '../types/actions.types';

export const startNodeAction = (body: StartNodeActionBody) =>
  post<ActionStarted, StartNodeActionBody>('actions:startNodeAction', body);

export const startServiceAction = (body: StartServiceActionBody) =>
  post<StartServiceActionResponse, StartServiceActionBody>(
    'actions:startServiceAction',
    body
  );

export const getAction = (actionId: string) =>
  get<GetActionResponse>(`actions/${actionId}`);

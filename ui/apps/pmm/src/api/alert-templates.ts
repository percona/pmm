import {
  CreateRulePayload,
  CreateTemplatePayload,
  DeleteTemplatePayload,
  ListTemplatesParams,
  ListTemplatesResponse,
  UpdateTemplatePayload,
} from 'types/alert-templates.types';
import { EmptyResponse } from 'types/util.types';
import { api } from './api';

export const listTemplates = async (
  params?: ListTemplatesParams
): Promise<ListTemplatesResponse> => {
  const res = await api.get<ListTemplatesResponse>('/alerting/templates', {
    params,
  });
  return res.data;
};

export const createTemplate = async (
  payload: CreateTemplatePayload
): Promise<EmptyResponse> => {
  const res = await api.post<EmptyResponse>('/alerting/templates', payload);
  return res.data;
};

export const updateTemplate = async ({
  name,
  yaml,
}: UpdateTemplatePayload): Promise<EmptyResponse> => {
  const res = await api.put<EmptyResponse>(
    `/alerting/templates/${encodeURIComponent(name)}`,
    { name, yaml }
  );
  return res.data;
};

export const deleteTemplate = async ({
  name,
}: DeleteTemplatePayload): Promise<EmptyResponse> => {
  const res = await api.delete<EmptyResponse>(
    `/alerting/templates/${encodeURIComponent(name)}`
  );
  return res.data;
};

export const createRuleFromTemplate = async (
  payload: CreateRulePayload
): Promise<EmptyResponse> => {
  const res = await api.post<EmptyResponse>('/alerting/rules', payload);
  return res.data;
};

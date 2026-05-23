import { api } from './api';

export interface LogParserPreset {
  id: string;
  name: string;
  description?: string;
  operatorYaml?: string;
  operator_yaml?: string;
  builtIn?: boolean;
  built_in?: boolean;
  usageCount?: number;
  usage_count?: number;
  createdAt?: string;
  created_at?: string;
  updatedAt?: string;
  updated_at?: string;
}

export function presetOperatorYaml(p: LogParserPreset): string {
  return (p.operatorYaml ?? p.operator_yaml ?? '').trim();
}

export function presetBuiltIn(p: LogParserPreset): boolean {
  return !!(p.builtIn ?? p.built_in);
}

export function presetUsageCount(p: LogParserPreset): number {
  return p.usageCount ?? p.usage_count ?? 0;
}

export async function listLogParserPresets(): Promise<LogParserPreset[]> {
  const res = await api.get<{ presets: LogParserPreset[] }>('/server/log-parser-presets');
  return res.data.presets ?? [];
}

export async function addLogParserPreset(body: {
  name: string;
  description?: string;
  operatorYaml: string;
}): Promise<LogParserPreset> {
  const res = await api.post<{ preset: LogParserPreset }>('/server/log-parser-presets', {
    name: body.name,
    description: body.description ?? '',
    operatorYaml: body.operatorYaml,
  });
  return res.data.preset;
}

export async function changeLogParserPreset(
  id: string,
  body: { description?: string; operatorYaml?: string }
): Promise<LogParserPreset> {
  const payload: Record<string, string> = {};
  if (body.description !== undefined) payload.description = body.description;
  if (body.operatorYaml !== undefined) payload.operatorYaml = body.operatorYaml;
  const res = await api.put<{ preset: LogParserPreset }>(`/server/log-parser-presets/${id}`, payload);
  return res.data.preset;
}

export async function removeLogParserPreset(id: string): Promise<void> {
  await api.delete(`/server/log-parser-presets/${id}`);
}

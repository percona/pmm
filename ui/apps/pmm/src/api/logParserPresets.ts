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

/** Remove common leading indent (String.trim only affects the first line). */
function dedentOperatorYaml(yaml: string): string {
  const lines = yaml.split('\n');
  let min: number | null = null;
  for (const line of lines) {
    if (!line.trim()) continue;
    const indent = line.length - line.trimStart().length;
    if (min === null || indent < min) min = indent;
  }
  if (!min) return yaml;
  return lines
    .map((line) => {
      if (!line.trim()) return '';
      return line.length >= min! ? line.slice(min!) : line.trimStart();
    })
    .join('\n');
}

/** Fix common copy/paste issues before sending operator YAML to the API. Keep in sync with Go NormalizeLogParserOperatorYAML. */
export function normalizeOperatorYaml(yaml: string): string {
  let s = yaml.replace(/\r\n/g, '\n').replace(/\r/g, '\n');
  s = s.replace(/[\u2018\u2019]/g, "'").replace(/[\u201c\u201d]/g, '"');
  if (!s.includes('\n') && s.includes('\\n')) {
    s = s.replace(/\\n/g, '\n');
  }
  s = s.replace(/' parse_from:/g, "'\n  parse_from:");
  s = s.replace(/' parse_to:/g, "'\n  parse_to:");
  s = s.replace(/' - type:/g, "'\n- type:");
  s = s.replace(/" parse_from:/g, '"\n  parse_from:');
  s = s.replace(/" parse_to:/g, '"\n  parse_to:');
  s = s.replace(/" - type:/g, '"\n- type:');
  s = s.replace(/^(\s*regex:\s+)(.+?\$\s*)parse_from:/gm, "$1'$2'\n  parse_from:");
  s = s.replace(/^(\s*regex:\s+)([^'"\n][^\n]*)$/gm, (line, prefix: string, value: string) => {
    const trimmed = value.trim();
    if (!trimmed || trimmed.startsWith("'") || trimmed.startsWith('"')) {
      return line;
    }
    if (trimmed.includes(':')) {
      return `${prefix}'${trimmed}'`;
    }
    return line;
  });
  s = dedentOperatorYaml(s);
  return s.trim();
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
  const res = await api.post<{ preset: LogParserPreset }>(
    '/server/log-parser-presets',
    {
      name: body.name,
      description: body.description ?? '',
      operatorYaml: normalizeOperatorYaml(body.operatorYaml),
    },
    { disableNotifications: true }
  );
  return res.data.preset;
}

export async function changeLogParserPreset(
  id: string,
  body: { description?: string; operatorYaml?: string }
): Promise<LogParserPreset> {
  const payload: Record<string, string> = {};
  if (body.description !== undefined) payload.description = body.description;
  if (body.operatorYaml !== undefined) payload.operatorYaml = normalizeOperatorYaml(body.operatorYaml);
  const res = await api.put<{ preset: LogParserPreset }>(
    `/server/log-parser-presets/${id}`,
    payload,
    { disableNotifications: true }
  );
  return res.data.preset;
}

export async function removeLogParserPreset(id: string): Promise<void> {
  await api.delete(`/server/log-parser-presets/${id}`);
}

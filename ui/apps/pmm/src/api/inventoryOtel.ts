import { api } from './api';

export interface OtelLogSource {
  path: string;
  preset: string;
}

export interface OtelCollectorAgent {
  agentId?: string;
  agent_id?: string;
  pmmAgentId?: string;
  pmm_agent_id?: string;
  disabled?: boolean;
  customLabels?: Record<string, string>;
  custom_labels?: Record<string, string>;
  status?: string;
}

export interface PmmAgentItem {
  agentId?: string;
  agent_id?: string;
  runsOnNodeId?: string;
  runs_on_node_id?: string;
}

export interface InventoryNodeItem {
  nodeId?: string;
  node_id?: string;
  name?: string;
}

export function agentId(a: OtelCollectorAgent): string {
  return a.agentId ?? a.agent_id ?? '';
}

export function pmmAgentId(a: OtelCollectorAgent): string {
  return a.pmmAgentId ?? a.pmm_agent_id ?? '';
}

export function collectorLabels(a: OtelCollectorAgent): Record<string, string> {
  return a.customLabels ?? a.custom_labels ?? {};
}

export function parseLogSourcesFromLabels(labels: Record<string, string>): OtelLogSource[] {
  const raw = labels.logSources ?? labels.log_sources;
  if (!raw) {
    const legacy = labels.logFilePaths ?? labels.log_file_paths;
    if (!legacy) return [];
    return legacy
      .split(',')
      .map((p) => p.trim())
      .filter(Boolean)
      .map((path) => ({ path, preset: 'raw' }));
  }
  try {
    const parsed = JSON.parse(raw) as OtelLogSource[];
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

export async function listOtelCollectors(): Promise<OtelCollectorAgent[]> {
  const res = await api.get<{ otelCollector?: OtelCollectorAgent[]; otel_collector?: OtelCollectorAgent[] }>(
    '/inventory/agents',
    { params: { agentType: 'AGENT_TYPE_OTEL_COLLECTOR' } }
  );
  return res.data.otelCollector ?? res.data.otel_collector ?? [];
}

export async function listPmmAgents(): Promise<PmmAgentItem[]> {
  const res = await api.get<{ pmmAgent?: PmmAgentItem[]; pmm_agent?: PmmAgentItem[] }>('/inventory/agents', {
    params: { agentType: 'AGENT_TYPE_PMM_AGENT' },
  });
  return res.data.pmmAgent ?? res.data.pmm_agent ?? [];
}

export async function listInventoryNodes(): Promise<InventoryNodeItem[]> {
  const res = await api.get<{
    generic?: InventoryNodeItem[];
    container?: InventoryNodeItem[];
  }>('/inventory/nodes');
  return [...(res.data.generic ?? []), ...(res.data.container ?? [])];
}

export async function changeOtelCollectorLogSources(
  agentId: string,
  logSources: OtelLogSource[]
): Promise<OtelCollectorAgent> {
  const res = await api.put<{ otelCollector?: OtelCollectorAgent; otel_collector?: OtelCollectorAgent }>(
    `/inventory/agents/${agentId}`,
    {
      otelCollector: {
        replaceLogSources: true,
        setLogSources: logSources.map((s) => ({
          path: s.path.trim(),
          preset: s.preset.trim() || 'raw',
        })),
      },
    }
  );
  const agent = res.data.otelCollector ?? res.data.otel_collector;
  if (!agent) throw new Error('Unexpected empty response from ChangeAgent');
  return agent;
}

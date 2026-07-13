import { api } from './api';
import type { AdreUsageEvent } from './adre';

export type InvestigationUsageEvent = Pick<
  AdreUsageEvent,
  | 'id'
  | 'feature'
  | 'model'
  | 'createdAt'
  | 'created_at'
  | 'totalTokens'
  | 'total_tokens'
  | 'cachedTokens'
  | 'cached_tokens'
  | 'totalCost'
  | 'total_cost'
>;

export interface InvestigationListItem {
  id: string;
  title: string;
  status: string;
  createdAt: string;
  updatedAt: string;
  /** Backend may return snake_case */
  created_at?: string;
  updated_at?: string;
  timeFrom?: string;
  timeTo?: string;
  sourceType?: string;
  source_type?: string;
  nodeName?: string;
  node_name?: string;
  serviceName?: string;
  service_name?: string;
}

export interface InvestigationBlock {
  id: string;
  investigationId: string;
  type: string;
  title: string;
  position: number;
  configJson?: Record<string, unknown>;
  dataJson?: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
}

export interface InvestigationEvidenceEntry {
  id: string;
  kind: string;
  claim: string;
  source_tool: string;
  source_ref: string;
  excerpt: string;
  time_range: string;
  verification: string;
}

export interface Investigation {
  id: string;
  title: string;
  status: string;
  severity: string;
  createdAt: string;
  updatedAt: string;
  createdBy: string;
  timeFrom: string;
  timeTo: string;
  summary: string;
  userRequest?: string;
  user_request?: string;
  summaryDetailed: string;
  rootCauseSummary: string;
  resolutionSummary: string;
  sourceType: string;
  sourceRef: string;
  nodeName?: string;
  serviceName?: string;
  clusterName?: string;
  servicenowTicketId?: string;
  servicenow_ticket_id?: string;
  servicenowTicketNumber?: string;
  servicenow_ticket_number?: string;
  holmesTotalTokens?: number;
  holmes_total_tokens?: number;
  holmesTotalCost?: number;
  holmes_total_cost?: number;
  holmesCallCount?: number;
  holmes_call_count?: number;
  confidence: 'high' | 'medium' | 'low';
  confidenceScore: number;
  confidenceRationale: string;
  evidence: InvestigationEvidenceEntry[];
  blocks?: InvestigationBlock[];
}

export interface InvestigationComment {
  id: string;
  investigationId: string;
  blockId?: string | null;
  anchorJson?: Record<string, unknown> | null;
  author: string;
  content: string;
  createdAt: string;
  updatedAt: string;
}

export interface InvestigationMessage {
  id: string;
  investigationId: string;
  role: string;
  content: string;
  toolName?: string;
  toolResultJson?: Record<string, unknown>;
  createdAt: string;
  model?: string;
  promptTokens?: number;
  prompt_tokens?: number;
  completionTokens?: number;
  completion_tokens?: number;
  totalTokens?: number;
  total_tokens?: number;
  cachedTokens?: number;
  cached_tokens?: number;
  totalCost?: number;
  total_cost?: number;
  holmesFeature?: string;
  holmes_feature?: string;
}

export interface CreateInvestigationBody {
  title: string;
  timeFrom?: string;
  timeTo?: string;
  sourceType?: string;
  sourceRef?: string;
  summary?: string;
  nodeName?: string;
  serviceName?: string;
  clusterName?: string;
  /** Full alert payload(s) when creating from alert; sent to backend for Holmes context. */
  alertSnapshot?: unknown;
}

export interface PatchInvestigationBody {
  title?: string;
  status?: string;
  summary?: string;
  summaryDetailed?: string;
  rootCauseSummary?: string;
  resolutionSummary?: string;
  severity?: string;
  timeFrom?: string;
  timeTo?: string;
}

export interface CreateBlockBody {
  type: string;
  title?: string;
  position?: number;
  configJson?: Record<string, unknown>;
  dataJson?: Record<string, unknown>;
}

export interface PatchBlockBody {
  type?: string;
  title?: string;
  position?: number;
  configJson?: Record<string, unknown>;
  dataJson?: Record<string, unknown>;
}

export interface CreateCommentBody {
  content: string;
  blockId?: string | null;
  anchorJson?: Record<string, unknown> | null;
  author?: string;
}

export const listInvestigations = async (params?: {
  status?: string;
  trigger?: 'auto' | 'manual';
  limit?: number;
  offset?: number;
  orderBy?: string;
  order?: 'asc' | 'desc';
}): Promise<InvestigationListItem[]> => {
  const res = await api.get<InvestigationListItem[]>('/investigations', {
    params: params
      ? {
          status: params.status,
          ...(params.trigger != null && { trigger: params.trigger }),
          limit: params.limit,
          offset: params.offset,
          ...(params.orderBy != null && { order_by: params.orderBy }),
          ...(params.order != null && { order: params.order }),
        }
      : undefined,
  });
  return res.data;
};

export const getInvestigation = async (id: string): Promise<Investigation> => {
  const res = await api.get<Investigation>(`/investigations/${id}`);
  return res.data;
};

export const createInvestigation = async (
  body: CreateInvestigationBody
): Promise<Investigation> => {
  const payload: Record<string, unknown> = {
    title: body.title,
    ...(body.timeFrom != null && { time_from: body.timeFrom }),
    ...(body.timeTo != null && { time_to: body.timeTo }),
    ...(body.sourceType != null && { source_type: body.sourceType }),
    ...(body.sourceRef != null && { source_ref: body.sourceRef }),
    ...(body.summary != null && { summary: body.summary }),
    ...(body.nodeName && { node_name: body.nodeName }),
    ...(body.serviceName && { service_name: body.serviceName }),
    ...(body.clusterName && { cluster_name: body.clusterName }),
    ...(body.alertSnapshot != null && { alert_snapshot: body.alertSnapshot }),
  };
  const res = await api.post<Investigation>('/investigations', payload);
  return res.data;
};

export const patchInvestigation = async (
  id: string,
  body: PatchInvestigationBody
): Promise<Investigation> => {
  const res = await api.patch<Investigation>(`/investigations/${id}`, body);
  return res.data;
};

export const deleteInvestigation = async (id: string): Promise<void> => {
  await api.delete(`/investigations/${id}`);
};

export const getInvestigationBlocks = async (
  id: string
): Promise<InvestigationBlock[]> => {
  const res = await api.get<InvestigationBlock[]>(`/investigations/${id}/blocks`);
  return res.data;
};

export const postInvestigationBlock = async (
  id: string,
  body: CreateBlockBody
): Promise<InvestigationBlock> => {
  const res = await api.post<InvestigationBlock>(
    `/investigations/${id}/blocks`,
    body
  );
  return res.data;
};

export const patchInvestigationBlock = async (
  investigationId: string,
  blockId: string,
  body: PatchBlockBody
): Promise<InvestigationBlock> => {
  const res = await api.patch<InvestigationBlock>(
    `/investigations/${investigationId}/blocks/${blockId}`,
    body
  );
  return res.data;
};

export const deleteInvestigationBlock = async (
  investigationId: string,
  blockId: string
): Promise<void> => {
  await api.delete(`/investigations/${investigationId}/blocks/${blockId}`);
};

export const getInvestigationComments = async (
  id: string,
  blockId?: string
): Promise<InvestigationComment[]> => {
  const res = await api.get<InvestigationComment[]>(
    `/investigations/${id}/comments`,
    { params: blockId ? { block_id: blockId } : undefined }
  );
  return res.data;
};

export const getInvestigationMessages = async (
  id: string,
  params?: { limit?: number; offset?: number }
): Promise<InvestigationMessage[]> => {
  const res = await api.get<InvestigationMessage[]>(
    `/investigations/${id}/messages`,
    { params: params ?? {} }
  );
  return res.data;
};

export interface InvestigationTimelineEvent {
  id: string;
  investigationId: string;
  /** API returns camelCase (eventTime) when using axios-case-converter */
  eventTime: string;
  type: string;
  title: string;
  description: string;
  source: string;
}

export const getInvestigationTimeline = async (
  id: string
): Promise<InvestigationTimelineEvent[]> => {
  const res = await api.get<InvestigationTimelineEvent[]>(
    `/investigations/${id}/timeline`
  );
  return res.data;
};

export const postInvestigationComment = async (
  id: string,
  body: CreateCommentBody
): Promise<InvestigationComment> => {
  const res = await api.post<InvestigationComment>(
    `/investigations/${id}/comments`,
    body
  );
  return res.data;
};

export interface ChatResponse {
  content: string;
}

export const postInvestigationChat = async (
  id: string,
  body: { message: string }
): Promise<ChatResponse> => {
  const res = await api.post<ChatResponse>(`/investigations/${id}/chat`, body);
  return res.data;
};

export const postInvestigationRun = async (id: string): Promise<ChatResponse> => {
  const res = await api.post<ChatResponse>(`/investigations/${id}/run`, {});
  return res.data;
};

/** URL for the PDF/print export page; open in a new window to print or save as PDF. */
export const getInvestigationExportPdfUrl = (id: string): string => {
  const base = typeof window !== 'undefined' ? window.location.origin : '';
  return `${base}/v1/investigations/${id}/export/pdf`;
};

export interface CreateServiceNowTicketResponse {
  success: boolean;
  ticket_id: string;
  ticket_number?: string;
  message: string;
}

export const createServiceNowTicket = async (
  id: string
): Promise<CreateServiceNowTicketResponse> => {
  const res = await api.post<CreateServiceNowTicketResponse>(
    `/investigations/${id}/servicenow`,
    {}
  );
  return res.data;
};

export const getInvestigationUsage = async (
  investigationId: string
): Promise<{ investigationId: string; events: InvestigationUsageEvent[] }> => {
  const res = await api.get<{ investigationId: string; events: InvestigationUsageEvent[] }>(
    `/investigations/${investigationId}/usage`
  );
  return res.data;
};

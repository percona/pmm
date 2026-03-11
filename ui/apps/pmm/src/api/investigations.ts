import { api } from './api';

export interface InvestigationListItem {
  id: string;
  title: string;
  status: string;
  createdAt: string;
  updatedAt: string;
  timeFrom?: string;
  timeTo?: string;
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
  summaryDetailed: string;
  rootCauseSummary: string;
  resolutionSummary: string;
  sourceType: string;
  sourceRef: string;
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
}

export interface CreateInvestigationBody {
  title: string;
  timeFrom?: string;
  timeTo?: string;
  sourceType?: string;
  sourceRef?: string;
  summary?: string;
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

export interface CreateCommentBody {
  content: string;
  blockId?: string | null;
  anchorJson?: Record<string, unknown> | null;
  author?: string;
}

export const listInvestigations = async (params?: {
  status?: string;
  limit?: number;
  offset?: number;
}): Promise<InvestigationListItem[]> => {
  const res = await api.get<InvestigationListItem[]>('/investigations', {
    params: params ? { status: params.status, limit: params.limit, offset: params.offset } : undefined,
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
  const res = await api.post<Investigation>('/investigations', body);
  return res.data;
};

export const patchInvestigation = async (
  id: string,
  body: PatchInvestigationBody
): Promise<Investigation> => {
  const res = await api.patch<Investigation>(`/investigations/${id}`, body);
  return res.data;
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

export enum AgentUpdateSeverity {
  UNSPECIFIED = 'UPDATE_SEVERITY_UNSPECIFIED',
  UNSUPPORTED = 'UPDATE_SEVERITY_UNSUPPORTED',
  UP_TO_DATE = 'UPDATE_SEVERITY_UP_TO_DATE',
  REQUIRED = 'UPDATE_SEVERITY_REQUIRED',
  CRITICAL = 'UPDATE_SEVERITY_CRITICAL',
}

export interface GetAgentVersionItem {
  agentId: string;
  version: string;
  nodeName: string;
  severity: AgentUpdateSeverity;
}

export interface GetAgentVersionsResponse {
  agentVersions: GetAgentVersionItem[];
}

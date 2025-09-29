export interface RealTimeQueryData {
  queryId: string;
  database: string;
  fingerprint: string;
  queryText?: string;
  timestamp: string;
  state: QueryState;
  clientHost?: string;
  currentExecutionTime: number;
  mongodb?: MongoDBFields;
  
  // Service metadata (may not be present in all responses)
  serviceId?: string;
  serviceName?: string;
  nodeId?: string;
  nodeName?: string;
  labels?: Record<string, string>;
}

export interface MongoDBFields {
  opid: number;
  secsRunning: number;
  operationType: string;
  namespace: string;
  planSummary?: string;
  blocking: boolean;
  currentOpRaw?: string; // Complete raw currentOp document as JSON
}

export enum QueryState {
  UNKNOWN = 'UNKNOWN',
  RUNNING = 'RUNNING', 
  WAITING = 'WAITING',
  FINISHED = 'FINISHED',
}

export interface RealTimeAnalyticsRequest {
  queries: RealTimeQueryData[];
}

export interface RealTimeDataResponse {
  queries: RealTimeQueryData[];
  totalCount?: number;
  hasMore?: boolean;
}

// Enhanced data structure with service metadata for UI consumption
export interface RealTimeServiceData {
  serviceId: string;
  serviceName: string;
  serviceType: string;
  nodeId: string;
  nodeName: string;
  address?: string;
  port?: number;
  labels: Record<string, string>;
  queries?: RealTimeQueryData[];
  isEnabled?: boolean;
  config?: RealTimeConfig;
  lastSeen?: string;
}

export interface RealTimeConfig {
  collectionIntervalSeconds: number;
  disableExamples: boolean;
}


export interface EnableRealTimeAnalyticsRequest {
  serviceId: string;
  config: RealTimeConfig;
}

export interface DisableRealTimeAnalyticsRequest {
  serviceId: string;
}

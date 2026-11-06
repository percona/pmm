// Response of Grafana's POST /api/ds/query: data frames keyed by query refId.
export interface DataFrameField {
  name?: string;
  type?: string;
  labels?: Record<string, string>;
}

export interface DataFrameJSON {
  schema?: {
    refId?: string;
    fields: DataFrameField[];
  };
  data?: {
    values: unknown[][];
  };
}

export interface DataQueryResult {
  status?: number;
  error?: string;
  frames?: DataFrameJSON[];
}

export interface DataQueryResponse {
  results: Record<string, DataQueryResult>;
}

export interface InstantQuery {
  refId: string;
  expr: string;
}

// One series of an instant query: its labels and current value.
export interface InstantSample {
  labels: Record<string, string>;
  value?: number;
}

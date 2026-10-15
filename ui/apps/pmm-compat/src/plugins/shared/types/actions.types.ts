export interface ActionStarted {
  action_id: string;
  pmm_agent_id: string;
}

export interface StartNodeActionBody {
  node_id: string;
}

export interface ServiceActionTarget {
  service_id: string;
}

export interface StartServiceActionBody {
  pt_mongodb_summary?: ServiceActionTarget;
  pt_mysql_summary?: ServiceActionTarget;
  pt_postgres_summary?: ServiceActionTarget;
}

export interface StartServiceActionResponse {
  pt_mongodb_summary?: ActionStarted;
  pt_mysql_summary?: ActionStarted;
  pt_postgres_summary?: ActionStarted;
}

export interface GetActionResponse {
  action_id: string;
  pmm_agent_id: string;
  output: string;
  done: boolean;
  error: string;
}

export interface ActionResult {
  value: string | null;
  error: string;
}

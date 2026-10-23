import type { OmInventoryRun, OmTopologyResponse } from '@pmm/plugins-om';
// The module, not the package index, which would load the whole Operations UI.
import { request } from '@pmm/plugins-om/src/api';

/** The fields of a bootstrap run the hub reads; the package does not export the full type. */
export interface OmBootstrapRunSummary {
  run_id: string;
  status: 'running' | 'succeeded' | 'failed' | 'rolled_back';
  hosts: Array<{ host: string }>;
  replica_set_name: string;
  started_at: string;
  finished_at?: string | null;
  environment?: string | null;
  cluster?: string | null;
}

/** `GET /v1/om/topology`: the latest snapshot of the MongoDB estate. */
export const getTopology = () => request<OmTopologyResponse>('/topology');

/** `GET /v1/om/inventory/runs`: host scans, newest first. */
export const getRuns = async (limit = 20): Promise<OmInventoryRun[]> => {
  const { runs } = await request<{ runs?: OmInventoryRun[] }>(
    `/inventory/runs?limit=${limit}`
  );

  return runs ?? [];
};

/** `GET /v1/om/inventory/bootstrap-runs`: MongoDB installs, newest first. */
export const getBootstrapRuns = async (
  limit = 20
): Promise<OmBootstrapRunSummary[]> => {
  const { runs } = await request<{ runs?: OmBootstrapRunSummary[] }>(
    `/inventory/bootstrap-runs?limit=${limit}`
  );

  return runs ?? [];
};

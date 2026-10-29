import { grafanaApi } from './api';

// PMM's provisioned "Metrics" datasource; the same uid on every PMM.
const METRICS_DATASOURCE_UID = 'PA58DA793C7250F1B';
// Only labelled clusters count: the Kubernetes dashboards ignore series without k8s_cluster_id.
const PRESENCE_QUERY = 'group(kube_node_info{k8s_cluster_id!=""})';

interface InstantQueryResponse {
  data: { result: unknown[] };
}

export const hasKubernetesData = async (): Promise<boolean> => {
  try {
    const res = await grafanaApi.get<InstantQueryResponse>(
      `/datasources/proxy/uid/${METRICS_DATASOURCE_UID}/api/v1/query`,
      { params: { query: PRESENCE_QUERY } }
    );
    return res.data.data.result.length > 0;
  } catch {
    return false;
  }
};

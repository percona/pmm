import { api } from './api';
import { MANAGEMENT_CREATE_NODE_INSTALL_TOKEN } from './managementEndpoints';

export interface CreateNodeInstallTokenResponse {
  token: string;
  expiresAt?: string;
  // The server also returns serviceAccountId (int64-as-string in JSON), but the
  // UI does not consume it today. Re-add the field if/when we expose a revoke
  // action — see managed/services/grafana/client.go::CreateNodeInstallToken.
}

/** Mints a short-lived Grafana token for PMM Client install (authenticated admin session). */
export async function createNodeInstallToken(
  technology: string,
  ttlSeconds = 0
): Promise<CreateNodeInstallTokenResponse> {
  const res = await api.post<CreateNodeInstallTokenResponse>(MANAGEMENT_CREATE_NODE_INSTALL_TOKEN, {
    ttlSeconds,
    technology,
  });
  return res.data;
}

import { api } from './api';
import { MANAGEMENT_CREATE_NODE_INSTALL_TOKEN } from './managementEndpoints';

export interface CreateNodeInstallTokenResponse {
  token: string;
  expiresAt?: string;
  serviceAccountId?: string;
}

/** Mints a short-lived Grafana token for PMM Client install (admin session). */
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

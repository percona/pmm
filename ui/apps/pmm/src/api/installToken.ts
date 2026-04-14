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
  return api.post<CreateNodeInstallTokenResponse, { ttlSeconds: number; technology: string }>(
    MANAGEMENT_CREATE_NODE_INSTALL_TOKEN,
    {
      ttlSeconds,
      technology,
    },
    true
  );
}

import { isAxiosError } from 'axios';
import { grafanaApi } from './api';

export interface CreateNodeInstallTokenResponse {
  token: string;
  expiresAt: string;
}

// Single source of truth for the install token's lifetime. The token only needs to
// stay valid long enough to run `pmm-admin config` once; the agent then receives a
// durable node service-account token from PMM Server (see RegisterNode), so a short
// TTL keeps an Admin-grade token from lingering in terminal scrollback. Exported so
// the UI can show the exact validity window without hardcoding it.
export const DEFAULT_TTL_SECONDS = 15 * 60;
export const DEFAULT_TTL_MINUTES = DEFAULT_TTL_SECONDS / 60;

const SUPPORTED_TECHNOLOGIES = new Set([
  'mysql',
  'postgresql',
  'mongodb',
  'valkey',
]);

// Shared SA per technology, created lazily on first use. Same naming scheme the
// removed backend endpoint used so previously-minted SAs are still reusable.
const SA_NAME_PREFIX = 'pmm-install-sa';
const TOKEN_NAME_PREFIX = 'pmm-install-st';

interface GrafanaServiceAccount {
  id: number;
  name: string;
}

interface GrafanaServiceAccountSearch {
  totalCount: number;
  serviceAccounts: GrafanaServiceAccount[];
}

interface GrafanaTokenResponse {
  id: number;
  name: string;
  key: string;
}

interface GrafanaServiceAccountToken {
  id: number;
  name: string;
  expiration: string | null;
  secondsUntilExpiration: number;
  hasExpired?: boolean;
}

const randomTokenSuffix = (): string => {
  if (typeof crypto?.randomUUID === 'function') {
    return crypto.randomUUID();
  }

  if (typeof crypto?.getRandomValues === 'function') {
    const bytes = new Uint8Array(16);
    crypto.getRandomValues(bytes);
    bytes[6] = (bytes[6] & 0x0f) | 0x40;
    bytes[8] = (bytes[8] & 0x3f) | 0x80;
    const hex = [...bytes].map((b) => b.toString(16).padStart(2, '0')).join('');
    return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
  }

  // Unlikely: both APIs are unavailable outside a secure context. Collision
  // resistance only needs to be good enough for Grafana token-name uniqueness.
  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;
}

/**
 * Mints a short-lived Grafana service-account token for a PMM Client install command.
 *
 * Implementation note: this calls Grafana's serviceaccounts API directly through the
 * `/graph/api/` reverse proxy. Grafana requires the **organization Admin** role for
 * these endpoints — Editor and Viewer users get 403 on search/create/token calls
 * (verified against `/graph/api/serviceaccounts/search` and `POST /graph/api/serviceaccounts`).
 */
export async function createNodeInstallToken(
  technology: string,
): Promise<CreateNodeInstallTokenResponse> {
  if (!SUPPORTED_TECHNOLOGIES.has(technology)) {
    throw new Error(`unsupported technology "${technology}"`);
  }

  const saName = `${SA_NAME_PREFIX}-${technology}`;

  let saId = await findServiceAccountIdByName(saName);
  if (saId === null) {
    saId = await createServiceAccount(saName);
  }

  await deleteExpiredTokens(saId);

  // UUID-suffixed token name keeps concurrent calls from colliding on Grafana's
  // per-SA unique-name constraint (Grafana returns 409 otherwise).
  const tokenName = `${TOKEN_NAME_PREFIX}-${technology}-${randomTokenSuffix()}`;
  const key = await mintToken(saId, tokenName, DEFAULT_TTL_SECONDS);

  return {
    token: key,
    expiresAt: new Date(Date.now() + DEFAULT_TTL_SECONDS * 1000).toISOString(),
  };
}

async function findServiceAccountIdByName(name: string): Promise<number | null> {
  // perpage keeps the exact-name match from falling off the first page when many
  // similarly-prefixed service accounts exist (Grafana's search is a substring match).
  const res = await grafanaApi.get<GrafanaServiceAccountSearch>(
    '/serviceaccounts/search',
    { params: { query: name, perpage: 100 } }
  );
  const match = res.data.serviceAccounts?.find((sa) => sa.name === name);
  return match ? match.id : null;
}

function isExpiredInstallToken(token: GrafanaServiceAccountToken): boolean {
  if (!token.name.startsWith(`${TOKEN_NAME_PREFIX}-`)) {
    return false;
  }

  if (token.hasExpired) {
    return true;
  }

  return token.expiration != null && token.secondsUntilExpiration <= 0;
}

async function deleteExpiredTokens(serviceAccountId: number): Promise<void> {
  let tokens: GrafanaServiceAccountToken[];
  try {
    const res = await grafanaApi.get<GrafanaServiceAccountToken[]>(
      `/serviceaccounts/${serviceAccountId}/tokens`
    );
    tokens = res.data;
  } catch {
    // Housekeeping only — don't block minting if Grafana won't list tokens.
    return;
  }

  const expired = tokens.filter(isExpiredInstallToken);
  const results = await Promise.allSettled(
    expired.map((token) =>
      grafanaApi.delete(
        `/serviceaccounts/${serviceAccountId}/tokens/${token.id}`
      )
    )
  );
  // Best-effort cleanup, but surface persistent failures so a broken delete path
  // (permissions, rate limiting) is at least visible instead of silently piling up.
  const failed = results.filter((r) => r.status === 'rejected').length;
  if (failed > 0) {
    // eslint-disable-next-line no-console
    console.warn(
      `installToken: failed to delete ${failed}/${expired.length} expired install token(s)`
    );
  }
}

async function createServiceAccount(name: string): Promise<number> {
  // Admin role is required for `pmm-admin config`/inventory writes in real PMM setups.
  try {
    const res = await grafanaApi.post<GrafanaServiceAccount>('/serviceaccounts', {
      name,
      role: 'Admin',
      isDisabled: false,
    });
    return res.data.id;
  } catch (error) {
    // A concurrent "Generate token" (or double-click) may have created the SA first;
    // Grafana enforces unique SA names and returns 409. Reuse the existing one instead
    // of failing the whole flow.
    if (isAxiosError(error) && error.response?.status === 409) {
      const existing = await findServiceAccountIdByName(name);
      if (existing !== null) {
        return existing;
      }
    }
    throw error;
  }
}

async function mintToken(
  serviceAccountId: number,
  tokenName: string,
  ttlSeconds: number
): Promise<string> {
  // Only `name` + `secondsToLive` — extra fields (`role`) have been observed to
  // make some Grafana versions ignore `secondsToLive` and fall back to a long default.
  const res = await grafanaApi.post<GrafanaTokenResponse>(
    `/serviceaccounts/${serviceAccountId}/tokens`,
    { name: tokenName, secondsToLive: ttlSeconds }
  );
  return res.data.key;
}

import { grafanaApi } from './api';

export interface CreateNodeInstallTokenResponse {
  token: string;
  expiresAt: string;
}

// Hard cap mirrors the previous server-side cap (15 min). Tokens longer than this
// shouldn't be in someone's terminal scrollback — re-run "Generate token" instead.
const MAX_TTL_SECONDS = 15 * 60;
const DEFAULT_TTL_SECONDS = 15 * 60;

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
 * `/graph/api/` reverse proxy. The user must already be authenticated as a Grafana
 * Admin (Grafana rejects the create/mint requests with 403 otherwise) — that's the
 * same trust boundary the old backend endpoint had, just one hop shorter.
 */
export async function createNodeInstallToken(
  technology: string,
  ttlSeconds = 0
): Promise<CreateNodeInstallTokenResponse> {
  if (!SUPPORTED_TECHNOLOGIES.has(technology)) {
    throw new Error(`unsupported technology "${technology}"`);
  }

  let ttl = ttlSeconds > 0 ? ttlSeconds : DEFAULT_TTL_SECONDS;
  if (ttl > MAX_TTL_SECONDS) {
    ttl = MAX_TTL_SECONDS;
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
  const key = await mintToken(saId, tokenName, ttl);

  return {
    token: key,
    expiresAt: new Date(Date.now() + ttl * 1000).toISOString(),
  };
}

async function findServiceAccountIdByName(name: string): Promise<number | null> {
  const res = await grafanaApi.get<GrafanaServiceAccountSearch>(
    '/serviceaccounts/search',
    { params: { query: name } }
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
  await Promise.allSettled(
    expired.map((token) =>
      grafanaApi.delete(
        `/serviceaccounts/${serviceAccountId}/tokens/${token.id}`
      )
    )
  );
}

async function createServiceAccount(name: string): Promise<number> {
  // Admin role is required for `pmm-admin config`/inventory writes in real PMM setups.
  const res = await grafanaApi.post<GrafanaServiceAccount>('/serviceaccounts', {
    name,
    role: 'Admin',
    isDisabled: false,
  });
  return res.data.id;
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

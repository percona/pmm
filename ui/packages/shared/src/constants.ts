/** Grafana's sub-path on PMM Server (it runs with serve_from_sub_path). */
export const GRAFANA_SUB_PATH = '/graph';

/**
 * Grafana routes that keep the /graph prefix instead of being pulled into the PMM UI shell: the
 * REST API, the image renderer and the auth/account pages.
 *
 * Single source for the redirect contract. The nginx exclusion regex in
 * build/ansible/roles/nginx/files/conf.d/pmm.conf is built from the same alternation and
 * nginxParity.test.ts fails the build if they drift.
 */
export const GRAFANA_DIRECT_PATH_SEGMENTS = [
  'api',
  'render',
  'login',
  'logout',
  'signup',
  'invite',
  'verify',
  'user/password/(send-reset-email|reset)',
  // Grafana navigates here as a top-level document when the session expiry lapses. The shell has
  // no such route, so pulling it in rotates the token inside the iframe while the address bar
  // stays pinned to the rotate URL.
  'user/auth-tokens/rotate',
] as const;

/** The alternation body shared verbatim with the nginx regex. */
export const GRAFANA_DIRECT_PATH_ALTERNATION =
  GRAFANA_DIRECT_PATH_SEGMENTS.join('|');

export const GRAFANA_DIRECT_PATH_PATTERN = new RegExp(
  `^${GRAFANA_SUB_PATH}/(${GRAFANA_DIRECT_PATH_ALTERNATION})(/|$)`
);

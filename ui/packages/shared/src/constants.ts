/**
 * Grafana routes that must keep the /graph prefix and never be pulled into the PMM UI shell: the
 * REST API, the image renderer and the auth/account pages. Kept in sync with the redirect
 * exclusion regex in build/ansible/roles/nginx/files/conf.d/pmm.conf (location /graph) — nginx
 * exempts these server-side, and neither the compat plugin nor the shell may undo that.
 *
 * Test it against a path only, never a path plus query string — nginx matches on $uri for the
 * same reason.
 */
export const GRAFANA_DIRECT_PATH_PATTERN =
  /^\/graph\/(api|render|login|logout|signup|invite|verify|user\/password\/(send-reset-email|reset))(\/|$)/;

/** Sample paths for the redirect contract, shared by every suite that asserts on it. */
export const GRAFANA_DIRECT_PATHS = [
  '/graph/api',
  '/graph/api/datasources',
  '/graph/render/d/some-dashboard',
  '/graph/login',
  '/graph/login/generic_oauth',
  '/graph/logout',
  '/graph/signup',
  '/graph/invite/abc123',
  '/graph/verify',
  '/graph/user/password/send-reset-email',
  '/graph/user/password/reset',
  '/graph/user/auth-tokens/rotate',
];

/** Shell-bound paths: superficially similar, but none of them bypass the shell. */
export const GRAFANA_SHELL_PATHS = [
  '/graph/apidocs',
  '/graph/logins',
  '/graph/d/node-cpu',
  '/graph/',
  '/graph',
  '/graph/user/password',
  '/graph/user/auth-tokens',
  '/pmm-ui/graph/d/node-cpu',
];

/**
 * nginx tests a percent-decoded, slash-collapsed $uri; window.location.pathname does neither, so
 * these must be recognised client-side too or the shell undoes the server-side exemption.
 */
export const GRAFANA_DIRECT_PATHS_OBFUSCATED = [
  '/graph/%6Cogin',
  '/graph/%61pi/datasources',
  '/graph/user/password/%72eset',
  '/graph/%75ser/auth-tokens/rotate',
  '/graph//login',
  '/graph///api/datasources',
];

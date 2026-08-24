import { GRAFANA_DIRECT_PATH_PATTERN } from './constants';

describe('GRAFANA_DIRECT_PATH_PATTERN', () => {
  // Must stay identical to the nginx exclusion regex in
  // build/ansible/roles/nginx/files/conf.d/pmm.conf (location /graph).
  it.each([
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
  ])('matches %s', (path) => {
    expect(GRAFANA_DIRECT_PATH_PATTERN.test(path)).toBe(true);
  });

  it.each([
    '/graph/apidocs',
    '/graph/logins',
    '/graph/d/node-cpu',
    '/graph/',
    '/graph',
    '/graph/user/password',
    '/pmm-ui/graph/d/node-cpu',
  ])('does not match %s', (path) => {
    expect(GRAFANA_DIRECT_PATH_PATTERN.test(path)).toBe(false);
  });
});

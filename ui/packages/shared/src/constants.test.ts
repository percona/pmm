import { isGrafanaDirectPath } from './constants';

describe('isGrafanaDirectPath', () => {
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
    expect(isGrafanaDirectPath(path)).toBe(true);
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
    expect(isGrafanaDirectPath(path)).toBe(false);
  });

  // nginx tests a percent-decoded, slash-collapsed $uri; window.location.pathname does neither, so
  // these must be recognised here too or the shell undoes the server-side exemption.
  it.each([
    '/graph/%6Cogin',
    '/graph/%61pi/datasources',
    '/graph/user/password/%72eset',
    '/graph//login',
    '/graph///api/datasources',
  ])('matches the normalised form of %s', (path) => {
    expect(isGrafanaDirectPath(path)).toBe(true);
  });

  it('returns a verdict rather than throwing on a malformed escape', () => {
    expect(isGrafanaDirectPath('/graph/%zz')).toBe(false);
    expect(isGrafanaDirectPath('/graph/%')).toBe(false);
  });
});

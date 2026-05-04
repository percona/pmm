import { describe, expect, it } from 'vitest';
import { getLocationUrl, isGrafanaLoginPath } from './grafana.utils';

describe('getLocationUrl', () => {
  it('maps Grafana-relative /login to /graph/login', () => {
    expect(
      getLocationUrl({
        pathname: '/login',
        search: '',
        hash: '',
        key: 'k',
      })
    ).toBe('/graph/login');
  });

  it('keeps /graph/login when Grafana already uses full graph path', () => {
    expect(
      getLocationUrl({
        pathname: '/graph/login',
        search: '',
        hash: '',
        key: 'k',
      })
    ).toBe('/graph/login');
  });

  it('strips /pmm-ui prefix when present', () => {
    expect(
      getLocationUrl({
        pathname: '/pmm-ui/graph/d/x',
        search: '?a=1',
        hash: '',
        key: 'k',
      })
    ).toBe('/graph/d/x?a=1');
  });
});

describe('isGrafanaLoginPath', () => {
  it('detects /graph/login', () => {
    expect(isGrafanaLoginPath('/graph/login')).toBe(true);
    expect(isGrafanaLoginPath('/graph/login?redirect=/')).toBe(true);
  });

  it('rejects other graph routes', () => {
    expect(isGrafanaLoginPath('/graph/d/pmm-home')).toBe(false);
    expect(isGrafanaLoginPath('/graph/admin/users')).toBe(false);
  });
});

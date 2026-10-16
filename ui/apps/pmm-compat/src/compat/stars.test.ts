import { watchDashboardStars } from './stars';

describe('watchDashboardStars', () => {
  const originalFetch = window.fetch;
  let ok = true;

  beforeEach(() => {
    ok = true;
    window.fetch = jest.fn(async () => ({ ok }) as Response);
  });

  afterEach(() => {
    window.fetch = originalFetch;
  });

  it('reports a dashboard being starred or unstarred', async () => {
    const onChange = jest.fn();
    watchDashboardStars(onChange);

    await window.fetch('/graph/api/user/stars/dashboard/uid/abc', {
      method: 'POST',
    });
    await window.fetch('/graph/api/user/stars/dashboard/uid/abc', {
      method: 'DELETE',
    });

    expect(onChange).toHaveBeenCalledTimes(2);
  });

  it('reports a star saved through the collections API', async () => {
    const onChange = jest.fn();
    watchDashboardStars(onChange);

    await window.fetch(
      '/graph/apis/collections.grafana.app/v1alpha1/namespaces/default/stars/user-1/update/dashboard.grafana.app/Dashboard/abc',
      { method: 'PUT' }
    );

    expect(onChange).toHaveBeenCalledTimes(1);
  });

  it('ignores reading the stars, other requests and failed ones', async () => {
    const onChange = jest.fn();
    watchDashboardStars(onChange);

    await window.fetch('/graph/api/user/stars');
    await window.fetch('/graph/api/dashboards/uid/abc', { method: 'POST' });
    ok = false;
    await window.fetch('/graph/api/user/stars/dashboard/uid/abc', {
      method: 'POST',
    });

    expect(onChange).not.toHaveBeenCalled();
  });
});

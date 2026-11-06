import { isMigratedPage } from './grafana.utils';

describe('isMigratedPage', () => {
  it('matches pages moved out of Grafana', () => {
    expect(isMigratedPage('/graph/settings/advanced')).toBe(true);
    expect(isMigratedPage('/graph/pmm-dump')).toBe(true);
    expect(isMigratedPage('/graph/pmm-dump/new')).toBe(true);
  });

  it('keeps other Grafana pages in the iframe', () => {
    expect(isMigratedPage('/graph/d/pmm-home')).toBe(false);
  });
});

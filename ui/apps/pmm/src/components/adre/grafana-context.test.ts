import { describe, expect, it } from 'vitest';
import {
  buildGrafanaDashboardContext,
  parseGrafanaLocation,
  stripPmmUiPrefix,
} from './grafana-context';

describe('stripPmmUiPrefix', () => {
  it('removes /pmm-ui prefix', () => {
    expect(stripPmmUiPrefix('/pmm-ui/graph/d/mysql-home/foo')).toBe('/graph/d/mysql-home/foo');
  });

  it('leaves graph paths unchanged when no prefix', () => {
    expect(stripPmmUiPrefix('/graph/d/pmm-home/home')).toBe('/graph/d/pmm-home/home');
  });
});

describe('parseGrafanaLocation', () => {
  it('returns null for non-Grafana paths', () => {
    expect(parseGrafanaLocation('/adre', '')).toBeNull();
    expect(parseGrafanaLocation('/pmm-ui/investigations', '')).toBeNull();
  });

  it('parses dashboard uid and viewPanel', () => {
    const p = parseGrafanaLocation(
      '/pmm-ui/graph/d/mysql-instance-summary/instance',
      '?viewPanel=panel-92&from=now-1h&to=now&var-service_name=mysql-mysql',
    );
    expect(p).not.toBeNull();
    expect(p!.kind).toBe('dashboard');
    expect(p!.dashboardUid).toBe('mysql-instance-summary');
    expect(p!.searchParams.get('viewPanel')).toBe('panel-92');
    expect(p!.searchParams.get('from')).toBe('now-1h');
    expect(p!.searchParams.get('var-service_name')).toBe('mysql-mysql');
  });

  it('parses explore', () => {
    const p = parseGrafanaLocation('/graph/explore', '?left=%7B%22datasource%22%3A%22Prometheus%22%7D');
    expect(p!.kind).toBe('explore');
    expect(p!.dashboardUid).toBeNull();
  });

  it('parses d-solo path uid', () => {
    const p = parseGrafanaLocation('/graph/d-solo/mysql-innodb', '?panelId=38');
    expect(p!.kind).toBe('d-solo');
    expect(p!.dashboardUid).toBe('mysql-innodb');
  });

  it('accepts lowercase viewpanel', () => {
    const p = parseGrafanaLocation('/graph/d/x/y', '?viewpanel=panel-1');
    expect(p!.searchParams.get('viewpanel')).toBe('panel-1');
  });
});

describe('buildGrafanaDashboardContext', () => {
  const origin = 'https://pmm.example';

  it('returns empty string off Grafana', () => {
    expect(buildGrafanaDashboardContext('/settings', '', origin, null)).toBe('');
  });

  it('includes full URL, uid, viewPanel, vars, and rules', () => {
    const ctx = buildGrafanaDashboardContext(
      '/pmm-ui/graph/d/mysql-innodb/details',
      '?viewPanel=panel-38&from=now-3h&to=now&var-node_name=mysql',
      origin,
      'MySQL / InnoDB - Grafana',
    );
    expect(ctx).toContain(`${origin}/pmm-ui/graph/d/mysql-innodb/details`);
    expect(ctx).toContain('Dashboard UID: mysql-innodb');
    expect(ctx).toContain('Focused panel (viewPanel): panel-38');
    expect(ctx).toContain('Grafana tab / document title: MySQL / InnoDB - Grafana');
    expect(ctx).toContain('var-node_name=mysql');
    expect(ctx).toContain('Rules for this context:');
    expect(ctx).toContain('do NOT claim a specific panel ID');
  });

  it('states no focused panel when viewPanel missing on dashboard', () => {
    const ctx = buildGrafanaDashboardContext(
      '/graph/d/mysql-instance-summary/slug',
      '?from=now-1h&to=now',
      origin,
      null,
    );
    expect(ctx).toContain('not set in the URL');
    expect(ctx).not.toContain('Focused panel (viewPanel):');
  });

  it('marks explore in context', () => {
    const ctx = buildGrafanaDashboardContext('/graph/explore', '', origin, null);
    expect(ctx).toContain('Path kind: explore');
    expect(ctx).toContain('Grafana Explore');
  });
});

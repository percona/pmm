import {
  GRAFANA_DIRECT_PATHS,
  GRAFANA_DIRECT_PATHS_OBFUSCATED,
  GRAFANA_SHELL_PATHS,
} from './grafanaDirectPaths.fixtures';
import { GRAFANA_DIRECT_PATH_SEGMENTS, GRAFANA_SUB_PATH } from './constants';
import { isGrafanaDirectPath } from './utils';

describe('isGrafanaDirectPath', () => {
  it.each(GRAFANA_DIRECT_PATHS)('matches %s', (path) => {
    expect(isGrafanaDirectPath(path)).toBe(true);
  });

  it.each(GRAFANA_SHELL_PATHS)('does not match %s', (path) => {
    expect(isGrafanaDirectPath(path)).toBe(false);
  });

  it.each(GRAFANA_DIRECT_PATHS_OBFUSCATED)(
    'matches the normalised form of %s',
    (path) => {
      expect(isGrafanaDirectPath(path)).toBe(true);
    }
  );

  it('returns a verdict rather than throwing on a malformed escape', () => {
    expect(isGrafanaDirectPath('/graph/%zz')).toBe(false);
    expect(isGrafanaDirectPath('/graph/%')).toBe(false);
  });

  // The fixtures are hand-written, so a new segment could otherwise land untested everywhere.
  it.each([...GRAFANA_DIRECT_PATH_SEGMENTS])(
    'has at least one fixture covering the %s segment',
    (segment) => {
      const matcher = new RegExp(`^${GRAFANA_SUB_PATH}/(${segment})(/|$)`);

      expect(GRAFANA_DIRECT_PATHS.some((path) => matcher.test(path))).toBe(
        true
      );
    }
  );
});

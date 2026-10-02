import { readFileSync } from 'fs';
import { resolve } from 'path';
import { GRAFANA_DIRECT_PATH_ALTERNATION, GRAFANA_SUB_PATH } from './constants';

// __dirname, not process.cwd(): the latter varies with how the suite is invoked.
const PMM_CONF = resolve(
  __dirname,
  '../../../..',
  'build/ansible/roles/nginx/files/conf.d/pmm.conf'
);

// Per test, not at module scope, so a missing pmm.conf fails an assertion rather than collection.
const confLines = () =>
  readFileSync(PMM_CONF, 'utf8')
    .split('\n')
    .map((line) => line.trim());

describe('nginx redirect exclusions', () => {
  // When nginx and the shell disagree, one lets a route through and the other bounces it back,
  // and the two redirect at each other.
  it('are built from the same alternation as GRAFANA_DIRECT_PATH_SEGMENTS', () => {
    // Whole line, not a substring: a substring search also accepts the directive commented out.
    expect(confLines()).toContain(
      `if ($uri ~ "^${GRAFANA_SUB_PATH}/(${GRAFANA_DIRECT_PATH_ALTERNATION})(/|$)") {`
    );
  });

  it('reads a pmm.conf that actually has the redirect in it', () => {
    expect(
      confLines().some((line) => line.startsWith('set $redirect_to_pmm_ui'))
    ).toBe(true);
  });
});

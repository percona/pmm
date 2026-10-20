import { screen } from '@testing-library/react';
import { renderHub } from '../../testUtils';
import { minutesAgo, testDoor, testView } from '../hub.testFixtures';
import type { HubStatus } from '../hub.types';
import { HubStatusLine } from './HubStatusLine';

const STATUS = testView().status;

// The sentence after the icon, whose accessible name also counts as text content.
const sentence = () => screen.getByTestId('hub-status-line').lastElementChild;

const renderLine = (status: Partial<HubStatus> = {}, loading = false) =>
  renderHub(
    <HubStatusLine
      status={{ ...STATUS, ...status }}
      name="MongoDB"
      unit={['cluster', 'clusters']}
      loading={loading}
    />
  );

describe('HubStatusLine', () => {
  it('makes each figure a door whose name starts with its visible text', () => {
    renderLine({
      doors: {
        clusters: testDoor('Clusters', '/mongodb#clusters', {
          context: 'in prod',
        }),
        attention: testDoor('Needs attention', '/mongodb#needs-attention'),
        backup: testDoor('Last good backup', '/graph/backup/inventory', {
          context: 'of billing',
        }),
        patches: testDoor('Behind on patches', '/mongodb?patches=behind'),
        alerts: testDoor('Firing alerts', '/alerting/status'),
      },
    });

    expect(sentence()).toHaveTextContent(
      /^3 clusters in prod · 3 need attention · last good backup 6h ago of billing$/
    );
    expect(
      screen.getByRole('link', { name: '3 clusters in prod' })
    ).toHaveAttribute('href', '/mongodb#clusters');
    expect(
      screen.getByRole('link', { name: '3 need attention' })
    ).toHaveAttribute('href', '/mongodb#needs-attention');
    expect(
      screen.getByRole('link', { name: 'last good backup 6h ago of billing' })
    ).toHaveAttribute('href', '/graph/backup/inventory');
  });

  it('says none need attention as plain text', () => {
    renderLine({ state: 'healthy', needAttention: 0 });

    expect(screen.getByText(/none need attention/)).toBeInTheDocument();
    expect(
      screen.queryByRole('link', { name: /attention/ })
    ).not.toBeInTheDocument();
  });

  it('says one needs attention in the singular', () => {
    renderLine({ needAttention: 1 });

    expect(
      screen.getByRole('link', { name: '1 needs attention' })
    ).toBeInTheDocument();
  });

  it('reads a dash, not zero, when metrics could not be asked', () => {
    renderLine({ needAttention: null });

    expect(screen.getByText('— need attention')).toHaveAttribute(
      'aria-hidden',
      'true'
    );
    expect(
      screen.getByText('Could not check what needs attention')
    ).toBeInTheDocument();
  });

  it('reads dashes, not zero, when the inventory could not be asked', () => {
    renderLine({
      state: 'noData',
      clusters: null,
      needAttention: null,
      lastGoodBackup: 'unknown',
    });

    expect(sentence()).toHaveTextContent(/^— clusters/);
    expect(
      screen.getByText('Could not count the clusters')
    ).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: /clusters/ })).toBeNull();
    expect(screen.queryByText(/none need attention/)).toBeNull();
  });

  it('offers to set up a backup when none is and the reader can', () => {
    renderHub(
      <HubStatusLine
        status={{
          ...STATUS,
          lastGoodBackup: 'notSetUp',
          doors: {
            ...STATUS.doors,
            backup: testDoor('Set up a backup', '/graph/backup/new'),
          },
        }}
        name="MongoDB"
        unit={['cluster', 'clusters']}
        canSetUpBackup
      />
    );

    expect(
      screen.getByRole('link', { name: 'set up a backup' })
    ).toHaveAttribute('href', '/graph/backup/new');
  });

  it.each<[HubStatus['lastGoodBackup'], string]>([
    ['none', 'no good backup yet'],
    ['notSetUp', 'no backups set up'],
    ['featureOff', 'backup management is off'],
    ['unknown', 'last good backup unknown'],
  ])('describes a %s backup state', (lastGoodBackup, text) => {
    renderLine({ lastGoodBackup });

    expect(screen.getByTestId('hub-status-line')).toHaveTextContent(text);
  });

  it('counts the rows no cluster accounts for, and links them', () => {
    renderLine({
      state: 'healthy',
      needAttention: 0,
      otherIssues: 2,
      attentionSeverity: 'warning',
    });

    expect(sentence()).toHaveTextContent(
      /^3 clusters · none need attention · 2 other issues · last good backup/
    );
    expect(
      screen.getByRole('link', { name: '2 other issues' })
    ).toHaveAttribute('href', '/mongodb#needs-attention');
  });

  it('names the estate’s health on its icon, so it is not color alone', () => {
    renderLine({ state: 'degraded', attentionSeverity: 'warning' });

    expect(screen.getByRole('img', { name: 'Degraded' })).toBeInTheDocument();
  });

  it('takes the icon from a row worse than the clusters’ health', () => {
    renderLine({
      state: 'healthy',
      needAttention: 1,
      attentionSeverity: 'error',
    });

    expect(screen.getByRole('img', { name: 'Error' })).toBeInTheDocument();
    expect(screen.queryByRole('img', { name: 'Healthy' })).toBeNull();
  });

  it('names an estate whose worst cluster is upgrading on an info icon', () => {
    renderLine({
      state: 'upgrading',
      needAttention: 0,
      attentionSeverity: undefined,
    });

    expect(screen.getByRole('img', { name: 'Upgrading' })).toBeInTheDocument();
  });

  it('links the clusters behind on patches and says how many the check could not answer for', () => {
    renderLine({ patches: { behind: 2, unknown: 1 } });

    expect(
      screen.getByRole('link', { name: '2 behind on patches' })
    ).toHaveAttribute('href', '/mongodb?patches=behind#clusters');
    expect(sentence()).toHaveTextContent('2 behind on patches, 1 unknown');
  });

  it('links the firing alerts, says when none fire, and reads a dash when they could not be read', () => {
    const { unmount } = renderLine({ alerts: 3 });

    expect(
      screen.getByRole('link', { name: '3 alerts firing' })
    ).toHaveAttribute('href', '/alerting/status');
    unmount();

    const second = renderLine({ alerts: 1 });

    expect(screen.getByRole('link', { name: '1 alert firing' })).toBeVisible();
    second.unmount();

    const third = renderLine({ alerts: 0 });

    expect(sentence()).toHaveTextContent('no alerts firing');
    third.unmount();

    renderLine({ alerts: null });
    expect(sentence()).toHaveTextContent('Could not read the firing alerts');
  });

  it('says none are behind as plain text, and nothing while no check runs', () => {
    const { unmount } = renderLine({ patches: { behind: 0, unknown: 0 } });

    expect(sentence()).toHaveTextContent('none behind on patches');
    expect(screen.queryByRole('link', { name: /patches/ })).toBeNull();
    unmount();

    renderLine({ patches: undefined });
    expect(sentence()).not.toHaveTextContent('patches');
  });

  it('keeps the health icon when no row is worse', () => {
    renderLine({ state: 'down', attentionSeverity: 'warning' });

    expect(screen.getByRole('img', { name: 'Down' })).toBeInTheDocument();
  });

  it('says nothing is monitored on an empty estate, with no doors', () => {
    renderLine({ state: 'empty', clusters: 0, needAttention: 0 });

    expect(screen.getByTestId('hub-status-line')).toHaveTextContent(
      'No MongoDB monitored yet'
    );
    expect(screen.queryByRole('link')).not.toBeInTheDocument();
  });

  it('shows skeletons instead of figures while loading', () => {
    renderLine({ checkedAt: minutesAgo(0) }, true);

    expect(screen.queryByRole('link')).not.toBeInTheDocument();
    expect(
      screen
        .getByTestId('hub-status-line')
        .querySelectorAll('.MuiSkeleton-root')
    ).toHaveLength(2);
  });
});

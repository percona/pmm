import { fireEvent, screen, within } from '@testing-library/react';
import { closestTo, renderHub } from '../../testUtils';
import {
  attentionItem,
  minutesAgo,
  READY,
  testDoor,
} from '../hub.testFixtures';
import type { HubAttentionItem, HubSectionState } from '../hub.types';
import { NeedsAttentionSection } from './NeedsAttentionSection';

const ITEMS: HubAttentionItem[] = [
  attentionItem('billing', {
    kind: 'noPrimary',
    severity: 'error',
    title: "billing has no primary, it can't take writes",
    detail: '2 of 3 members down',
    where: 'billing · prod',
    since: minutesAgo(18),
    door: testDoor('Open ReplSet summary', '/graph/d/replset', {
      context: 'for billing',
    }),
  }),
  attentionItem('orders-shard2-m2', {
    severity: 'error',
    title: 'orders-shard2-m2 is down',
    since: minutesAgo(12),
    source: 'alert',
    secondaryDoor: testDoor(
      'Alert',
      '/alerting/status?f.serviceName=orders-shard2-m2',
      {
        context: 'for orders-shard2-m2',
      }
    ),
  }),
  attentionItem('reports', { title: 'PMM gets no data from reports' }),
  attentionItem('backup', { title: 'orders nightly backup failed' }),
  attentionItem('pitr', { title: 'billing point-in-time recovery stopped' }),
  attentionItem('oplog', { title: 'orders-shard2 oplog window is 3h' }),
  attentionItem('restarted', { severity: 'info', title: 'Restarted' }),
];

const renderSection = (
  items: HubAttentionItem[] = ITEMS,
  state: HubSectionState = READY,
  onRetry = vi.fn()
) =>
  renderHub(
    <NeedsAttentionSection
      items={items}
      state={state}
      door={testDoor('Alerts', '/alerting/status')}
      checkedAt={minutesAgo(0)}
      onRetry={onRetry}
    />
  );

const rowTitles = (list: HTMLElement) =>
  within(list)
    .getAllByRole('listitem')
    .map((row) => row.querySelector('.MuiListItemText-primary')?.textContent);

describe('NeedsAttentionSection', () => {
  it('is a section named by its h2, with the count and the Alerts door', () => {
    renderSection();

    const section = screen.getByRole('region', { name: 'Needs attention' });

    expect(
      within(section).getByRole('heading', {
        level: 2,
        name: 'Needs attention',
      })
    ).toBeInTheDocument();
    expect(within(section).getByText('7')).toHaveTextContent('7 items');
    expect(
      within(section).getByRole('link', { name: 'Alerts' })
    ).toHaveAttribute('href', '/alerting/status');
  });

  it('shows five rows and folds the rest behind Show more', () => {
    renderSection();

    const [visible] = screen.getAllByRole('list');

    expect(rowTitles(visible)).toEqual([
      "billing has no primary, it can't take writes",
      'orders-shard2-m2 is down',
      'PMM gets no data from reports',
      'orders nightly backup failed',
      'billing point-in-time recovery stopped',
    ]);

    const more = screen.getByRole('button', { name: 'Show 2 more' });

    expect(more).toHaveAttribute('aria-expanded', 'false');

    const rest = document.getElementById(
      more.getAttribute('aria-controls') ?? ''
    );

    if (!rest) {
      throw new Error('Show more controls nothing');
    }

    expect(within(rest).getByText('Restarted')).toBeInTheDocument();

    fireEvent.click(more);

    expect(screen.getByRole('button', { name: 'Show fewer' })).toHaveAttribute(
      'aria-expanded',
      'true'
    );
  });

  it('never folds an error away', () => {
    const errors = Array.from({ length: 6 }, (_, index) =>
      attentionItem(`down-${index}`, { severity: 'error' })
    );

    renderSection([...errors, attentionItem('lag')]);

    const [visible] = screen.getAllByRole('list');

    expect(within(visible).getAllByRole('listitem')).toHaveLength(6);
    expect(
      screen.getByRole('button', { name: 'Show 1 more' })
    ).toBeInTheDocument();
  });

  it('gives a merged alert row its own Alert door', () => {
    renderSection();

    const row = closestTo(screen.getByText('orders-shard2-m2 is down'), 'li');

    expect(
      within(row).getByRole('link', {
        name: 'Open Instance summary for orders-shard2-m2',
      })
    ).toBeInTheDocument();
    expect(
      within(row).getByRole('link', { name: 'Alert for orders-shard2-m2' })
    ).toHaveAttribute(
      'href',
      '/alerting/status?f.serviceName=orders-shard2-m2'
    );
  });

  it('names each severity in words and puts the doors after the text', () => {
    renderSection();

    const row = closestTo(
      screen.getByText("billing has no primary, it can't take writes"),
      'li'
    );

    expect(within(row).getByRole('img', { name: 'Error' })).toBeInTheDocument();
    expect(row).toHaveTextContent('2 of 3 members down · billing · prod · 18m');

    const text = within(row).getByText(
      '2 of 3 members down · billing · prod · 18m'
    );
    const door = within(row).getByRole('link');

    expect(
      text.compareDocumentPosition(door) & Node.DOCUMENT_POSITION_FOLLOWING
    ).toBeTruthy();
  });

  it('shows one calm line and no list when nothing needs attention', () => {
    renderSection([]);

    expect(
      screen.getByText(/^Nothing needs attention · checked/)
    ).toBeInTheDocument();
    expect(screen.queryByRole('list')).not.toBeInTheDocument();
  });

  it('never shows the calm line while loading', () => {
    renderSection([], { status: 'loading' });

    expect(
      screen.queryByText(/Nothing needs attention/)
    ).not.toBeInTheDocument();
  });

  it('fails on its own with a warning and Retry', () => {
    const onRetry = vi.fn();

    renderSection(
      [],
      { status: 'error', message: 'Could not read metrics.' },
      onRetry
    );

    const alert = screen.getByRole('alert');

    expect(alert).toHaveTextContent('Could not read metrics.');
    expect(alert).toHaveClass('MuiAlert-standardWarning');

    fireEvent.click(within(alert).getByRole('button', { name: 'Retry' }));

    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it('keeps its rows under the warning when only a supplementary source failed', () => {
    renderSection(ITEMS.slice(0, 2), {
      status: 'error',
      source: 'alerts',
      message: 'Firing alerts could not be read, so none is listed here.',
      partial: true,
    });

    expect(screen.getByRole('alert')).toHaveTextContent(
      'Firing alerts could not be read'
    );
    expect(
      screen.getByText("billing has no primary, it can't take writes")
    ).toBeInTheDocument();
  });

  it('shows no calm line under a warning, with nothing else to list', () => {
    renderSection([], {
      status: 'error',
      source: 'alerts',
      message: 'Firing alerts could not be read, so none is listed here.',
      partial: true,
    });

    expect(screen.getByRole('alert')).toBeInTheDocument();
    expect(screen.queryByText(/Nothing needs attention/)).toBeNull();
  });
});

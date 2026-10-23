import { capAttention, higherSeverity, rankAttention } from './hub.attention';
import type { HubAttentionItem, HubSeverity } from './hub.types';

const item = (
  id: string,
  severity: HubSeverity,
  kind: string,
  since?: string
): HubAttentionItem => ({
  id,
  kind,
  severity,
  title: id,
  where: 'prod',
  ...(since ? { since } : {}),
  door: { label: 'Open', to: '/x' },
});

const ORDER = ['noPrimary', 'memberDown', 'behind', 'alert'];

describe('rankAttention', () => {
  it('ranks by severity first', () => {
    const ranked = rankAttention(
      [
        item('a', 'info', 'noPrimary'),
        item('b', 'warning', 'noPrimary'),
        item('c', 'error', 'alert'),
      ],
      ORDER
    );

    expect(ranked.map(({ id }) => id)).toEqual(['c', 'b', 'a']);
  });

  it('then by the kind order', () => {
    const ranked = rankAttention(
      [item('a', 'warning', 'alert'), item('b', 'warning', 'memberDown')],
      ORDER
    );

    expect(ranked.map(({ id }) => id)).toEqual(['b', 'a']);
  });

  it('puts kinds outside the order last', () => {
    const ranked = rankAttention(
      [item('a', 'warning', 'unknown'), item('b', 'warning', 'alert')],
      ORDER
    );

    expect(ranked.map(({ id }) => id)).toEqual(['b', 'a']);
  });

  it('then the longest-broken first, undated last', () => {
    const ranked = rankAttention(
      [
        item('a', 'warning', 'behind'),
        item('b', 'warning', 'behind', '2026-10-19T10:00:00Z'),
        item('c', 'warning', 'behind', '2026-10-19T09:00:00Z'),
      ],
      ORDER
    );

    expect(ranked.map(({ id }) => id)).toEqual(['c', 'b', 'a']);
  });

  it('then by id, so equal rows keep a stable order across refetches', () => {
    const items = [item('b', 'error', 'alert'), item('a', 'error', 'alert')];

    expect(rankAttention(items, ORDER).map(({ id }) => id)).toEqual(['a', 'b']);
    expect(
      rankAttention([...items].reverse(), ORDER).map(({ id }) => id)
    ).toEqual(['a', 'b']);
  });

  it('does not change its input', () => {
    const items = [item('b', 'info', 'alert'), item('a', 'error', 'alert')];

    rankAttention(items, ORDER);

    expect(items.map(({ id }) => id)).toEqual(['b', 'a']);
  });
});

describe('capAttention', () => {
  it('shows up to the cap and hides the rest', () => {
    const items = ['a', 'b', 'c', 'd'].map((id) =>
      item(id, 'warning', 'alert')
    );
    const { visible, hidden } = capAttention(items, 3);

    expect(visible.map(({ id }) => id)).toEqual(['a', 'b', 'c']);
    expect(hidden.map(({ id }) => id)).toEqual(['d']);
  });

  it('never folds an error away, even past the cap', () => {
    const items = [
      ...['e1', 'e2', 'e3'].map((id) => item(id, 'error', 'noPrimary')),
      item('w1', 'warning', 'alert'),
    ];
    const { visible, hidden } = capAttention(items, 2);

    expect(visible.map(({ id }) => id)).toEqual(['e1', 'e2', 'e3']);
    expect(hidden.map(({ id }) => id)).toEqual(['w1']);
  });

  it('keeps errors visible wherever they sit', () => {
    const items = [
      item('w1', 'warning', 'alert'),
      item('w2', 'warning', 'alert'),
      item('e1', 'error', 'alert'),
    ];

    expect(capAttention(items, 2).visible.map(({ id }) => id)).toEqual([
      'w1',
      'e1',
    ]);
  });

  it('fills the remaining room after errors', () => {
    const items = [
      item('e1', 'error', 'noPrimary'),
      item('e2', 'error', 'memberDown'),
      ...['w1', 'w2', 'w3', 'w4'].map((id) => item(id, 'warning', 'alert')),
    ];
    const { visible, hidden } = capAttention(items, 5);

    expect(visible).toHaveLength(5);
    expect(hidden.map(({ id }) => id)).toEqual(['w4']);
  });

  it('shows everything under the cap', () => {
    const items = [item('a', 'info', 'alert')];

    expect(capAttention(items, 5)).toEqual({ visible: items, hidden: [] });
  });
});

describe('higherSeverity', () => {
  it('picks the more severe', () => {
    expect(higherSeverity('warning', 'error')).toBe('error');
    expect(higherSeverity('error', 'info')).toBe('error');
    expect(higherSeverity('info', 'warning')).toBe('warning');
    expect(higherSeverity('info', 'info')).toBe('info');
  });
});

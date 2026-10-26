import {
  attentionInEnvironment,
  attentionIssues,
  higherSeverity,
  rankAttention,
  sortNeedsAttention,
} from './hub.attention';
import { attentionItem } from './hub.testFixtures';
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

describe('sortNeedsAttention', () => {
  it('puts critical first, then errors, then warnings, keeping each severity in its order', () => {
    const sorted = sortNeedsAttention([
      attentionItem('lag'),
      attentionItem('backups', { severity: 'error' }),
      attentionItem('oplog'),
      attentionItem('no primary', { severity: 'critical' }),
      attentionItem('down', { severity: 'error' }),
    ]);

    expect(sorted.map(({ title }) => title)).toEqual([
      'no primary',
      'backups',
      'down',
      'lag',
      'oplog',
    ]);
  });
});

describe('attentionIssues', () => {
  it('adds up the issues the rows stand for, not the rows', () => {
    expect(
      attentionIssues([
        attentionItem('lag', { issueCount: 7 }),
        attentionItem('down'),
      ])
    ).toBe(8);
  });
});

describe('attentionInEnvironment', () => {
  const rows = [
    attentionItem('prod only'),
    attentionItem('both', { environment: 'prod, staging' }),
    attentionItem('dev only', { environment: 'dev' }),
  ];

  it('keeps every row without a filter', () => {
    expect(attentionInEnvironment(rows, null)).toHaveLength(3);
  });

  it('keeps a row that spans environments in each of them', () => {
    expect(
      attentionInEnvironment(rows, 'staging').map(({ title }) => title)
    ).toEqual(['both']);
    expect(
      attentionInEnvironment(rows, 'prod').map(({ title }) => title)
    ).toEqual(['prod only', 'both']);
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

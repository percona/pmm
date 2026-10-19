import {
  canonicalKey,
  carriesVariables,
  normalizeText,
  techLabel,
  tokenize,
} from './search.text';

describe('search.text', () => {
  it.each([
    ['Médias', 'medias'],
    ['Real-time queries', 'real time queries'],
    ['mysql_summary', 'mysql summary'],
    ['  PXC/Galera   cluster ', 'pxc galera cluster'],
    ['Ｑｕｅｒｙ', 'query'],
  ])('normalizes %s', (text, expected) => {
    expect(normalizeText(text)).toBe(expected);
  });

  it('tokenizes into at most six words', () => {
    expect(tokenize('a-b c_d e f g h i')).toEqual([
      'a',
      'b',
      'c',
      'd',
      'e',
      'f',
    ]);
    expect(tokenize('  -- ')).toEqual([]);
  });

  it.each([
    ['/graph/d/mysql-innodb/mysql-innodb-details', 'dashboard:mysql-innodb'],
    ['/graph/d/mysql-innodb?var-service_name=a', 'dashboard:mysql-innodb'],
    ['/graph/d/pmm-ha/pmm-ha#panel', 'dashboard:pmm-ha'],
    ['/settings/', '/settings'],
    ['/graph/', '/graph'],
    ['/updates?x=1#y', '/updates'],
    ['/', '/'],
  ])('keys %s as %s', (url, key) => {
    expect(canonicalKey(url)).toBe(key);
  });

  it('has no key without a url', () => {
    expect(canonicalKey(undefined)).toBeUndefined();
    expect(canonicalKey('')).toBeUndefined();
  });

  it('carries variables only to plain dashboard paths', () => {
    expect(carriesVariables('/graph/d/node-cpu/cpu-utilization-details')).toBe(
      true
    );
    expect(carriesVariables('/graph/d/pmm-ha/x?var-namespace=a')).toBe(false);
    expect(carriesVariables('/graph/dashboards')).toBe(false);
    expect(carriesVariables(undefined)).toBe(false);
  });

  it.each([
    ['MySQL', 'Summary', 'MySQL summary'],
    ['MySQL', 'InnoDB details', 'MySQL InnoDB details'],
    ['MySQL', 'Group replication', 'MySQL group replication'],
    ['MongoDB', 'Oplog', 'MongoDB oplog'],
    ['Valkey', 'Slow Log', 'Valkey slow log'],
    ['Valkey', 'Cluster Details', 'Valkey cluster details'],
    ['PostgreSQL', 'Patroni', 'PostgreSQL Patroni'],
    ['Operating system', 'NUMA', 'Operating system NUMA'],
    ['MySQL', 'MySQL table details', 'MySQL table details'],
  ])('labels %s %s as %s', (tech, child, expected) => {
    expect(techLabel(tech, child)).toBe(expected);
  });
});

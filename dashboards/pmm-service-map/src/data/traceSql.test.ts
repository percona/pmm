import {
  buildNodeTraceServiceWhereClause,
  expandOtelServiceNameCandidates,
  expandTraceSearchTokens,
  podSegmentOrdinalVariants,
} from './traceSql';

describe('podSegmentOrdinalVariants', () => {
  it('strips repeated trailing -<digits> ordinals', () => {
    expect(podSegmentOrdinalVariants('pmm-ha-pg-db-instance1-2v9k-0')).toEqual([
      'pmm-ha-pg-db-instance1-2v9k-0',
      'pmm-ha-pg-db-instance1-2v9k',
    ]);
    expect(podSegmentOrdinalVariants('pmm-ha-pmmdb-0-0-0')).toEqual([
      'pmm-ha-pmmdb-0-0-0',
      'pmm-ha-pmmdb-0-0',
      'pmm-ha-pmmdb-0',
      'pmm-ha-pmmdb',
    ]);
  });
});

describe('expandOtelServiceNameCandidates', () => {
  it('adds OTLP pod-level alias without last ordinal (verified vs ClickHouse ServiceName)', () => {
    const c = expandOtelServiceNameCandidates([
      '/k8s/demo/pmm-ha-pg-db-instance1-2v9k-0/database',
    ]);
    expect(c).toContain('/k8s/demo/pmm-ha-pg-db-instance1-2v9k-0/database');
    expect(c).toContain('/k8s/demo/pmm-ha-pg-db-instance1-2v9k/database');
    expect(c).toContain('/k8s/demo/pmm-ha-pg-db-instance1-2v9k');
  });

  it('keeps non-k8s ids as-is', () => {
    expect(expandOtelServiceNameCandidates(['10.0.0.1:443'])).toEqual(['10.0.0.1:443']);
  });
});

describe('expandTraceSearchTokens', () => {
  it('adds k8s path segments for OTLP substring match', () => {
    const t = expandTraceSearchTokens(['/k8s/demo/pmm-ha-2/pmm-ha']);
    expect(t).toContain('/k8s/demo/pmm-ha-2/pmm-ha');
    expect(t).toContain('pmm-ha-2/pmm-ha');
    expect(t).toContain('pmm-ha-2');
    expect(t).toContain('pmm-ha');
  });
});

describe('buildNodeTraceServiceWhereClause', () => {
  it('combines ServiceName IN with position fallbacks', () => {
    const expanded = expandOtelServiceNameCandidates(['/k8s/demo/ns/pod-0/c']);
    const tokens = expandTraceSearchTokens(expanded);
    const w = buildNodeTraceServiceWhereClause(expanded, tokens);
    expect(w).toContain('ServiceName IN (');
    expect(w).toContain('position(lower(coalesce(ServiceName');
  });
});

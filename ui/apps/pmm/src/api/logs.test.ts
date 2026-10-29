import {
  findLogQuery,
  getLogFacets,
  getLogPatterns,
  searchLogs,
  severityClass,
  severityLabel,
} from './logs';

const { post } = vi.hoisted(() => ({ post: vi.fn() }));

vi.mock('./api', () => ({ api: { post } }));

describe('severityClass', () => {
  it('maps OpenTelemetry severity numbers to classes', () => {
    expect(severityClass(21)).toBe('error');
    expect(severityClass(17)).toBe('error');
    expect(severityClass(13)).toBe('warning');
    expect(severityClass(16)).toBe('warning');
    expect(severityClass(9)).toBe('info');
    expect(severityClass(5)).toBe('other');
    expect(severityClass(0)).toBe('other');
  });
});

describe('severityLabel', () => {
  it('names OpenTelemetry severity ranges whatever the source wrote', () => {
    expect(severityLabel(1)).toBe('TRACE');
    expect(severityLabel(5)).toBe('DEBUG');
    expect(severityLabel(9)).toBe('INFO');
    expect(severityLabel(12)).toBe('INFO');
    expect(severityLabel(13)).toBe('WARN');
    expect(severityLabel(17)).toBe('ERROR');
    expect(severityLabel(21)).toBe('FATAL');
  });

  it('has no label for an unset severity', () => {
    expect(severityLabel(0)).toBe('');
  });
});

describe('searchLogs', () => {
  it('maps the snake_case response and keeps attribute keys as sent', async () => {
    post.mockResolvedValue({
      data: {
        lines: [
          {
            time: '2026-10-17T10:00:00Z',
            node_name: 'pmm-server',
            source: 'pmm-managed',
            preset: 'pmm_managed',
            file_name: 'pmm-managed.log',
            severity_number: 17,
            severity_text: 'error',
            body: 'boom',
            body_truncated: true,
            attributes: {
              'log.file.name': 'pmm-managed.log',
              request_id: 'r1',
            },
            pattern_hash: '18446744073709551615',
          },
        ],
        next_cursor: 'c1',
      },
    });

    const res = await searchLogs(
      {
        start: '2026-10-17T09:00:00Z',
        end: '2026-10-17T10:00:00Z',
        minSeverity: 13,
        maxSeverity: 16,
        excludeNodeNames: ['pmm-server'],
        patternHash: '42',
        excludePatternHashes: ['7', '9'],
      },
      'c0',
      100
    );

    expect(post).toHaveBeenCalledWith(
      '/logs:search',
      {
        filter: {
          start: '2026-10-17T09:00:00Z',
          end: '2026-10-17T10:00:00Z',
          min_severity: 13,
          max_severity: 16,
          exclude_node_names: ['pmm-server'],
          pattern_hash: '42',
          exclude_pattern_hashes: ['7', '9'],
        },
        cursor: 'c0',
        limit: 100,
      },
      expect.objectContaining({ disableNotifications: true })
    );
    expect(res).toEqual({
      lines: [
        {
          time: '2026-10-17T10:00:00Z',
          nodeName: 'pmm-server',
          source: 'pmm-managed',
          preset: 'pmm_managed',
          fileName: 'pmm-managed.log',
          severityNumber: 17,
          severityText: 'error',
          body: 'boom',
          bodyTruncated: true,
          attributes: { 'log.file.name': 'pmm-managed.log', request_id: 'r1' },
          patternHash: '18446744073709551615',
        },
      ],
      nextCursor: 'c1',
    });
  });
});

describe('getLogPatterns', () => {
  it('maps the hash, the previous count and the top sources', async () => {
    post.mockResolvedValue({
      data: {
        patterns: [
          {
            pattern: 'deadlock detected',
            count: '12',
            previous_count: '4',
            pattern_hash: '18446744073709551615',
            top_sources: ['pg1'],
            first_seen: '2026-10-17T10:00:00Z',
            last_seen: '2026-10-17T10:05:00Z',
          },
          {
            pattern: 'checkpoint',
            first_seen: '2026-10-17T10:00:00Z',
            last_seen: '2026-10-17T10:05:00Z',
          },
        ],
      },
    });

    const [first, second] = await getLogPatterns({
      start: '2026-10-17T09:00:00Z',
      end: '2026-10-17T10:00:00Z',
    });

    expect(first).toMatchObject({
      pattern: 'deadlock detected',
      count: 12,
      previousCount: 4,
      patternHash: '18446744073709551615',
      topSources: ['pg1'],
    });
    expect(second).toMatchObject({
      count: 0,
      previousCount: 0,
      patternHash: '',
      topSources: [],
    });
  });
});

describe('findLogQuery', () => {
  it('sends the line and maps the found query', async () => {
    post.mockResolvedValue({
      data: {
        query_id: 'q1',
        service_name: 'pg-tcp',
        service_id: 's1',
        fingerprint: 'SELECT $1',
      },
    });

    const query = await findLogQuery({
      time: '2026-10-17T10:30:00.000Z',
      nodeName: 'pg-node',
      source: 'pg-node',
      preset: 'postgres',
      fileName: 'postgresql.log',
      severityNumber: 9,
      severityText: 'LOG',
      body: 'statement: SELECT 1',
      bodyTruncated: false,
      attributes: {},
      patternHash: '',
    });

    expect(post).toHaveBeenCalledWith(
      '/logs:findQuery',
      {
        time: '2026-10-17T10:30:00.000Z',
        node_name: 'pg-node',
        body: 'statement: SELECT 1',
      },
      expect.anything()
    );
    expect(query).toEqual({
      queryId: 'q1',
      serviceName: 'pg-tcp',
      serviceId: 's1',
      fingerprint: 'SELECT $1',
    });
  });

  it("sends a MongoDB line's attr", async () => {
    post.mockResolvedValue({ data: {} });
    await findLogQuery({
      time: '2026-10-17T10:30:00.000Z',
      nodeName: 'rs101',
      source: 'rs101',
      preset: 'mongodb',
      fileName: 'mongod.log',
      severityNumber: 9,
      severityText: 'I',
      body: 'Slow query',
      bodyTruncated: false,
      attributes: { attr: '{"type":"update"}', c: 'WRITE' },
      patternHash: '',
    });
    expect(post).toHaveBeenCalledWith(
      '/logs:findQuery',
      {
        time: '2026-10-17T10:30:00.000Z',
        node_name: 'rs101',
        body: 'Slow query',
        attributes: { attr: '{"type":"update"}' },
      },
      expect.anything()
    );
  });

  it('has an empty query id when QAN has none', async () => {
    post.mockResolvedValue({ data: {} });
    const query = await findLogQuery({
      time: '2026-10-17T10:30:00.000Z',
      nodeName: 'pg-node',
      source: '',
      preset: 'postgres',
      fileName: '',
      severityNumber: 9,
      severityText: '',
      body: 'statement: SELECT 1',
      bodyTruncated: false,
      attributes: {},
      patternHash: '',
    });
    expect(query.queryId).toBe('');
  });
});

describe('getLogFacets', () => {
  it('sends the service labels and maps their facets', async () => {
    post.mockResolvedValue({
      data: {
        node_names: [{ value: 'rs101', count: '40' }],
        clusters: [{ value: 'sharded', count: '40' }],
        environments: [{ value: 'prod', count: '7' }],
        replication_sets: [{ value: 'rs1', count: '40' }],
      },
    });

    const facets = await getLogFacets({
      start: '2026-10-17T09:00:00Z',
      end: '2026-10-17T10:00:00Z',
      clusters: ['sharded'],
      environments: ['prod'],
      replicationSets: ['rs1'],
    });

    expect(post).toHaveBeenCalledWith(
      '/logs:getFacets',
      {
        filter: {
          start: '2026-10-17T09:00:00Z',
          end: '2026-10-17T10:00:00Z',
          clusters: ['sharded'],
          environments: ['prod'],
          replication_sets: ['rs1'],
        },
      },
      expect.anything()
    );
    expect(facets).toEqual({
      nodeNames: [{ value: 'rs101', count: 40 }],
      sources: [],
      presets: [],
      clusters: [{ value: 'sharded', count: 40 }],
      environments: [{ value: 'prod', count: 7 }],
      replicationSets: [{ value: 'rs1', count: 40 }],
    });
  });
});

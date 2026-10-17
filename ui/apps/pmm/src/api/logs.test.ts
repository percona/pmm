import { searchLogs, severityClass } from './logs';

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
        },
      ],
      nextCursor: 'c1',
    });
  });
});

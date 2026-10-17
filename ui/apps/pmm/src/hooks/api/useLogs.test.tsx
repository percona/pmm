import { act, renderHook, waitFor } from '@testing-library/react';
import { ReactNode } from 'react';
import { LogLine } from 'api/logs';
import { wrapWithQueryProvider } from 'utils/testUtils';
import { useLogSearch } from './useLogs';

const { searchLogs } = vi.hoisted(() => ({ searchLogs: vi.fn() }));

vi.mock('api/logs', () => ({
  searchLogs,
  getLogVolume: vi.fn(),
  getLogPatterns: vi.fn(),
  getLogContext: vi.fn(),
  getLogFacets: vi.fn(),
}));

const filter = { start: '2026-10-17T10:00:00Z', end: '2026-10-17T11:00:00Z' };

const line = (body: string): LogLine => ({
  time: '2026-10-17T10:30:00Z',
  nodeName: 'pmm-server',
  source: 'pmm-managed',
  preset: 'pmm_managed',
  fileName: 'pmm-managed.log',
  severityNumber: 9,
  severityText: 'info',
  body,
  bodyTruncated: false,
  attributes: {},
});

const wrapper = ({ children }: { children: ReactNode }) =>
  wrapWithQueryProvider(children);

describe('useLogSearch', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('fetches the next page with nextCursor and stops when it is absent', async () => {
    searchLogs
      .mockResolvedValueOnce({ lines: [line('a')], nextCursor: 'c1' })
      .mockResolvedValueOnce({ lines: [line('b')] });

    const { result } = renderHook(() => useLogSearch(filter), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(searchLogs).toHaveBeenCalledWith(filter, undefined);
    expect(result.current.hasNextPage).toBe(true);

    await act(() => result.current.fetchNextPage());

    await waitFor(() => expect(result.current.data?.pages).toHaveLength(2));
    expect(searchLogs).toHaveBeenLastCalledWith(filter, 'c1');
    expect(result.current.hasNextPage).toBe(false);
  });
});

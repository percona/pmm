import React, { FC, PropsWithChildren } from 'react';
import { act, renderHook } from '@testing-library/react';
import { CancelToken } from 'axios';
import { QueryAnalyticsProvider } from 'pmm-qan/panel/provider/provider';
import OverviewService from './Overview.service';
import { useOverviewTable } from './Overview.hooks';

jest.mock('shared/components/helpers/notification-manager');
jest.mock('./Overview.service', () => ({
  __esModule: true,
  default: { getReport: jest.fn() },
}));
jest.mock('./components/MetricColumns/MetricColumns', () => ({
  getOverviewColumn: (key: string) => ({ accessor: key }),
}));
jest.mock('./components/DefaultColumns/DefaultColumns', () => ({
  getDefaultColumns: () => [],
}));

const getReportMock = OverviewService.getReport as jest.Mock;

function deferred<T>() {
  let resolve: (value: T) => void = () => {};
  let reject: (reason?: unknown) => void = () => {};
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });

  return { promise, resolve, reject };
}

const report = (fingerprint: string, total: number) => ({
  rows: [{ fingerprint, metrics: {} }],
  total_rows: total,
});

// Mirrors axios: a cancelled request rejects with a cancel error and never resolves.
const mockRequest = () => {
  const request = deferred<any>();

  getReportMock.mockImplementationOnce((_body, token: CancelToken) => {
    token.promise.then(request.reject);

    return request.promise;
  });

  return request;
};

const basePanelState = {
  labels: {},
  columns: ['load', 'num_queries'],
  pageNumber: 1,
  pageSize: 25,
  orderBy: '-load',
  from: 'now-1h',
  to: 'now',
  groupBy: 'queryid',
  dimensionSearchText: '',
};

let panelState = basePanelState;

const Wrapper: FC<PropsWithChildren<unknown>> = ({ children }) => (
  <QueryAnalyticsProvider.Provider value={{ panelState, contextActions: {} } as any}>
    {children}
  </QueryAnalyticsProvider.Provider>
);

describe('useOverviewTable::', () => {
  beforeEach(() => {
    panelState = basePanelState;
    getReportMock.mockReset();
  });

  it('shows the newest result after a superseded request is cancelled', async () => {
    const older = mockRequest();
    const newer = mockRequest();
    const setTotal = jest.fn();

    const { result, rerender } = renderHook(() => useOverviewTable(setTotal), { wrapper: Wrapper });

    panelState = { ...basePanelState, dimensionSearchText: 'pg_stat' };
    rerender();

    expect(getReportMock).toHaveBeenCalledTimes(2);
    expect(getReportMock.mock.calls[1][0].dimensionSearchText).toBe('pg_stat');
    await expect(older.promise).rejects.toHaveProperty('__CANCEL__', true);

    await act(async () => {
      newer.resolve(report('pg_stat_query', 22));
    });

    const [data, loading] = result.current;

    expect(data.rows).toEqual([{ fingerprint: 'pg_stat_query', metrics: {} }]);
    expect(setTotal).toHaveBeenCalledTimes(1);
    expect(setTotal).toHaveBeenCalledWith(22);
    expect(loading).toBe(false);
  });

  it('stays loading until the newest request finishes', async () => {
    mockRequest();
    const newer = mockRequest();

    const { result, rerender } = renderHook(() => useOverviewTable(jest.fn()), { wrapper: Wrapper });

    await act(async () => {
      panelState = { ...basePanelState, from: 'now-12h' };
      rerender();
    });

    expect(result.current[1]).toBe(true);
    expect(result.current[0].rows).toEqual([]);

    await act(async () => {
      newer.resolve(report('recent_query', 30));
    });

    expect(result.current[1]).toBe(false);
    expect(result.current[0].rows).toEqual([{ fingerprint: 'recent_query', metrics: {} }]);
  });

  it('stops loading when the request fails', async () => {
    const request = mockRequest();

    const { result } = renderHook(() => useOverviewTable(jest.fn()), { wrapper: Wrapper });

    expect(result.current[1]).toBe(true);

    await act(async () => {
      request.reject(new Error('server error'));
    });

    expect(result.current[1]).toBe(false);
  });

  it('cancels the in-flight request when parameters change or the table unmounts', () => {
    getReportMock.mockReturnValue(new Promise(() => {}));

    const { rerender, unmount } = renderHook(() => useOverviewTable(jest.fn()), { wrapper: Wrapper });
    const firstToken: CancelToken = getReportMock.mock.calls[0][1];

    expect(firstToken.reason).toBeUndefined();

    panelState = { ...basePanelState, orderBy: '-num_queries' };
    rerender();

    const secondToken: CancelToken = getReportMock.mock.calls[1][1];

    expect(firstToken.reason).toBeDefined();
    expect(secondToken.reason).toBeUndefined();

    unmount();

    expect(secondToken.reason).toBeDefined();
  });
});

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { act, renderHook, waitFor } from '@testing-library/react';
import messenger from 'lib/messenger';
import { wrapWithGrafana, wrapWithRouter } from 'utils/testUtils';
import { useLinkWithVariables } from './useLinkWithVariables';

vi.mock('lib/messenger', () => ({
  default: { sendMessageWithResult: vi.fn() },
}));

const CURRENT =
  '/graph/d/mysql-instance-summary/mysql-instance-summary?from=now-1h&to=now&var-service_name=mysql-1';
const TARGET = '/graph/d/node-cpu/cpu-utilization';

const renderWithUrl = (url: string, grafanaReady = true) =>
  renderHook(({ url }) => useLinkWithVariables(url), {
    initialProps: { url },
    wrapper: ({ children }) =>
      wrapWithRouter(wrapWithGrafana(children, { grafanaReady }), {
        initialEntries: ['/extensions/atw'],
      }),
  });

const render = (grafanaReady: boolean) =>
  renderHook(() => useLinkWithVariables(TARGET), {
    wrapper: ({ children }) =>
      wrapWithRouter(wrapWithGrafana(children, { grafanaReady }), {
        initialEntries: [CURRENT],
      }),
  });

describe('useLinkWithVariables', () => {
  beforeEach(() => {
    vi.mocked(messenger.sendMessageWithResult).mockReset();
  });

  it('should keep the time range before Grafana is ready', () => {
    const { result } = render(false);

    expect(result.current).toBe(`${TARGET}?from=now-1h&to=now`);
    expect(messenger.sendMessageWithResult).not.toHaveBeenCalled();
  });

  it('should keep the time range while Grafana resolves the link', () => {
    vi.mocked(messenger.sendMessageWithResult).mockReturnValue(
      new Promise(() => {})
    );
    const { result } = render(true);

    expect(result.current).toBe(`${TARGET}?from=now-1h&to=now`);
  });

  it('should keep the time range when Grafana returns the link unexpanded', async () => {
    vi.mocked(messenger.sendMessageWithResult).mockResolvedValue({
      url: '/d/node-cpu/cpu-utilization',
    });
    const { result } = render(true);

    await waitFor(() =>
      expect(messenger.sendMessageWithResult).toHaveBeenCalled()
    );
    // let the resolved link land in state
    await act(async () => {});
    expect(result.current).toBe(`${TARGET}?from=now-1h&to=now`);
  });

  it('should use the link Grafana expanded', async () => {
    vi.mocked(messenger.sendMessageWithResult).mockResolvedValue({
      url: '/d/node-cpu/cpu-utilization?from=now-1h&to=now&var-node_name=node-1',
    });
    const { result } = render(true);

    await waitFor(() =>
      expect(result.current).toBe(
        `${TARGET}?from=now-1h&to=now&var-node_name=node-1`
      )
    );
  });

  it('should follow a non-dashboard link that changes', () => {
    const { result, rerender } = renderWithUrl('/extensions/atw');

    expect(result.current).toBe('/extensions/atw');

    rerender({ url: '/extensions/task-manager' });

    expect(result.current).toBe('/extensions/task-manager');
    expect(messenger.sendMessageWithResult).not.toHaveBeenCalled();
  });

  it('should follow a link that changes after Grafana expanded the previous one', async () => {
    vi.mocked(messenger.sendMessageWithResult).mockResolvedValue({
      url: '/d/node-cpu/cpu-utilization?var-node_name=node-1',
    });
    const { result, rerender } = renderWithUrl(TARGET);

    await waitFor(() =>
      expect(result.current).toBe(`${TARGET}?var-node_name=node-1`)
    );

    rerender({ url: '/extensions/task-manager' });

    expect(result.current).toBe('/extensions/task-manager');
  });

  it('should ignore an expansion that arrives after the link changed', async () => {
    let resolve: (value: { url: string }) => void = () => {};
    vi.mocked(messenger.sendMessageWithResult).mockReturnValue(
      new Promise((r) => {
        resolve = r;
      })
    );
    const { result, rerender } = renderWithUrl(TARGET);

    await waitFor(() =>
      expect(messenger.sendMessageWithResult).toHaveBeenCalled()
    );
    rerender({ url: '/extensions/task-manager' });
    await act(async () => {
      resolve({ url: '/d/node-cpu/cpu-utilization?var-node_name=node-1' });
    });

    expect(result.current).toBe('/extensions/task-manager');
  });
});

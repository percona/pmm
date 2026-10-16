import { describe, it, expect, vi, beforeEach } from 'vitest';
import { act, renderHook, waitFor } from '@testing-library/react';
import type { ReactElement } from 'react';
import messenger from 'lib/messenger';
import { wrapWithGrafana, wrapWithRouter } from 'utils/testUtils';
import { useLinkWithVariables } from './useLinkWithVariables';

vi.mock('lib/messenger', () => ({
  default: { sendMessageWithResult: vi.fn() },
}));

const CURRENT =
  '/graph/d/mysql-instance-summary/mysql-instance-summary?from=now-1h&to=now&var-service_name=mysql-1';
const TARGET = '/graph/d/node-cpu/cpu-utilization';

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
});

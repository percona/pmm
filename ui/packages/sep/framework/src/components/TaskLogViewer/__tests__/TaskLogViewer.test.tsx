/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

import { act, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {
  afterEach,
  beforeEach,
  describe,
  expect,
  it,
  onTestFinished,
  vi,
} from 'vitest';
import {
  flushPromises,
  mockStreamFetch,
  type SseStreamHandle,
} from '../../../../tests/eventSourceStub';
import { QueryWrapper } from '../../../../tests/queryWrapper';
import { TaskLogViewer } from '../TaskLogViewer';

// Stub the log-viewer lib: real one depends on DOM APIs jsdom lacks.
vi.mock('@melloware/react-logviewer', () => ({
  LazyLog: ({ text }: { text: string }) => (
    <pre data-testid="log-output">{text}</pre>
  ),
}));

// Manual mock keeps axios out of the resolution graph.
let _tokenProvider: () => string | null = () => null;
vi.mock('@sep/api', () => ({
  setTokenProvider: (p: () => string | null) => {
    _tokenProvider = p;
  },
  getToken: () => _tokenProvider(),
  refreshAccessToken: vi.fn<() => Promise<string | null>>(),
  emitUnauthorized: vi.fn(),
  apiClient: { get: vi.fn(), defaults: {} },
  SEP_BASE_PATH: '/sep',
}));

describe('TaskLogViewer', () => {
  let mock: ReturnType<typeof mockStreamFetch>;

  beforeEach(() => {
    mock = mockStreamFetch();
    mock.install();
    _tokenProvider = () => 'test-token';
    globalThis.localStorage.clear();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.clearAllMocks();
    globalThis.localStorage.clear();
  });

  function streamUrlFor(id: string): string {
    return `/sep/stream-logs/${id}`;
  }

  function getHandle(id: string, callIndex = -1): SseStreamHandle {
    const matches = mock.pending.filter((handle) => {
      const path = handle.url.split('?')[0];
      return path === streamUrlFor(id);
    });
    const handle = callIndex < 0 ? matches.at(callIndex) : matches[callIndex];
    if (!handle) {
      throw new Error(`No stream handle for ${streamUrlFor(id)}`);
    }
    return handle;
  }

  function getEventHandle(id: string): SseStreamHandle {
    const handle = mock.pending
      .filter(
        (h) => h.url.split('?')[0] === `/sep/stream-logs/${id}/execution-events`
      )
      .at(-1);
    if (!handle) {
      throw new Error(`No execution-events stream handle for ${id}`);
    }
    return handle;
  }

  /** The primary stdout/stderr strip; the per-step strip is a second tablist. */
  function getPrimaryTabList() {
    return screen.getAllByRole('tablist')[0];
  }

  function getEventsButton() {
    return screen.getByRole('button', { name: /execution events/i });
  }

  function getTailSelect() {
    return screen.getByRole('combobox');
  }

  function queryTailSelect() {
    return screen.queryByRole('combobox');
  }

  function lines(count: number): string {
    return 'x\n'.repeat(count);
  }

  function fetchUrl(callIndex: number): string {
    const url = mock.fetchSpy.mock.calls[callIndex]?.[0];
    return typeof url === 'string' ? url : (url as URL).href;
  }

  function logFetchUrls(): string[] {
    return mock.fetchSpy.mock.calls
      .map((_, index) => fetchUrl(index))
      .filter((url) => /^\/sep\/stream-logs\/[^/?]+(\?|$)/.test(url));
  }

  it('renders accumulated stdout for the first step by default', async () => {
    render(
      <QueryWrapper>
        <TaskLogViewer taskHistoryId="7" taskStatus="RUNNING" />
      </QueryWrapper>
    );
    await flushPromises();

    const handle = getHandle('7');
    act(() => {
      handle.pushMessage({
        msg: 'line-1\n',
        step: 'setup',
        type: 'stdout',
        offset: 1,
      });
    });

    await waitFor(() =>
      expect(screen.getByTestId('log-output').textContent).toBe('line-1\n')
    );
  });

  it('omits tail param for running tasks', async () => {
    render(
      <QueryWrapper>
        <TaskLogViewer taskHistoryId="7" taskStatus="RUNNING" />
      </QueryWrapper>
    );
    await flushPromises();

    expect(logFetchUrls()[0]).toBe('/sep/stream-logs/7');
    expect(getTailSelect()).toHaveTextContent('Last 1000');
    expect(getTailSelect()).toHaveAttribute('aria-disabled', 'true');
  });

  it('requests tail=1000 by default for finished tasks', async () => {
    render(
      <QueryWrapper>
        <TaskLogViewer taskHistoryId="7" taskStatus="SUCCESS" />
      </QueryWrapper>
    );
    await flushPromises();

    expect(logFetchUrls()[0]).toBe('/sep/stream-logs/7?tail=1000');
    expect(getTailSelect()).toHaveTextContent('Last 1000');
  });

  it('restores the tail choice from localStorage for finished tasks', async () => {
    globalThis.localStorage.setItem('sep.taskLogViewer.tail', '5000');

    render(
      <QueryWrapper>
        <TaskLogViewer taskHistoryId="3" taskStatus="SUCCESS" />
      </QueryWrapper>
    );
    await flushPromises();

    expect(logFetchUrls()[0]).toBe('/sep/stream-logs/3?tail=5000');
    expect(getTailSelect()).toHaveTextContent('Last 5000');
  });

  it('omits tail param when All lines is selected for finished tasks', async () => {
    render(
      <QueryWrapper>
        <TaskLogViewer taskHistoryId="5" taskStatus="SUCCESS" />
      </QueryWrapper>
    );
    await flushPromises();
    expect(logFetchUrls()[0]).toBe('/sep/stream-logs/5?tail=1000');

    const user = userEvent.setup();
    await user.click(getTailSelect());
    await user.click(screen.getByRole('option', { name: /all lines/i }));
    await flushPromises();

    expect(logFetchUrls().at(-1)).toBe('/sep/stream-logs/5');
  });

  it('reconnects with a new tail when the line cap changes for finished tasks', async () => {
    render(
      <QueryWrapper>
        <TaskLogViewer taskHistoryId="8" taskStatus="SUCCESS" />
      </QueryWrapper>
    );
    await flushPromises();

    const user = userEvent.setup();
    await user.click(getTailSelect());
    await user.click(screen.getByRole('option', { name: 'Last 100' }));
    await flushPromises();

    expect(logFetchUrls().at(-1)).toBe('/sep/stream-logs/8?tail=100');
    expect(globalThis.localStorage.getItem('sep.taskLogViewer.tail')).toBe(
      '100'
    );
  });

  it('clears displayed logs when the line cap changes for finished tasks', async () => {
    render(
      <QueryWrapper>
        <TaskLogViewer taskHistoryId="11" taskStatus="SUCCESS" />
      </QueryWrapper>
    );
    await flushPromises();

    const firstHandle = getHandle('11');
    act(() => {
      firstHandle.pushMessage({
        msg: 'keep-me\n',
        step: 'setup',
        type: 'stdout',
        offset: 1,
      });
    });
    await waitFor(() =>
      expect(screen.getByTestId('log-output').textContent).toBe('keep-me\n')
    );

    const user = userEvent.setup();
    await user.click(getTailSelect());
    await user.click(screen.getByRole('option', { name: /last 5000/i }));
    await flushPromises();

    expect(screen.queryByTestId('log-output')).not.toBeInTheDocument();
    expect(screen.getByText(/no output yet/i)).toBeInTheDocument();
  });

  it('hides the line cap when a finished log is provably shorter than the smallest option', async () => {
    render(
      <QueryWrapper>
        <TaskLogViewer taskHistoryId="20" taskStatus="SUCCESS" />
      </QueryWrapper>
    );
    await flushPromises();

    const handle = getHandle('20');
    act(() => {
      handle.pushMessage({
        msg: lines(3),
        step: 'setup',
        type: 'stdout',
        offset: 1,
      });
      handle.pushMessage({
        msg: lines(2),
        step: 'setup',
        type: 'stderr',
        offset: 1,
      });
      handle.pushNamed('finish', { status: 'success' });
    });

    await waitFor(() => expect(queryTailSelect()).toBeNull());
    // Hiding the control leaves the stored choice alone for the next log.
    expect(
      globalThis.localStorage.getItem('sep.taskLogViewer.tail')
    ).toBeNull();
    // The request still carried the stored cap — size is unknown until it arrives.
    expect(logFetchUrls()[0]).toBe('/sep/stream-logs/20?tail=1000');
  });

  it('hides the line cap when a short finished log was fetched with All lines', async () => {
    globalThis.localStorage.setItem('sep.taskLogViewer.tail', 'all');

    render(
      <QueryWrapper>
        <TaskLogViewer taskHistoryId="21" taskStatus="SUCCESS" />
      </QueryWrapper>
    );
    await flushPromises();

    const handle = getHandle('21');
    act(() => {
      handle.pushMessage({
        msg: lines(4),
        step: 'setup',
        type: 'stdout',
        offset: 1,
      });
      handle.pushNamed('finish', { status: 'success' });
    });

    await waitFor(() => expect(queryTailSelect()).toBeNull());
  });

  it('keeps the line cap when a finished pane sits exactly at the requested cap', async () => {
    globalThis.localStorage.setItem('sep.taskLogViewer.tail', '100');

    render(
      <QueryWrapper>
        <TaskLogViewer taskHistoryId="22" taskStatus="SUCCESS" />
      </QueryWrapper>
    );
    await flushPromises();
    expect(logFetchUrls()[0]).toBe('/sep/stream-logs/22?tail=100');

    const handle = getHandle('22');
    act(() => {
      handle.pushMessage({
        msg: lines(100),
        step: 'setup',
        type: 'stdout',
        offset: 1,
      });
      handle.pushNamed('finish', { status: 'success' });
    });
    await waitFor(() =>
      expect(screen.getByTestId('log-output')).toBeInTheDocument()
    );

    // 100 lines under a tail=100 request may have been trimmed server-side.
    expect(getTailSelect()).toBeInTheDocument();
  });

  it('keeps the line cap when a finished log exceeds the smallest option', async () => {
    render(
      <QueryWrapper>
        <TaskLogViewer taskHistoryId="23" taskStatus="SUCCESS" />
      </QueryWrapper>
    );
    await flushPromises();

    const handle = getHandle('23');
    act(() => {
      handle.pushMessage({
        msg: lines(2),
        step: 'setup',
        type: 'stdout',
        offset: 1,
      });
      handle.pushMessage({
        msg: lines(150),
        step: 'build',
        type: 'stderr',
        offset: 1,
      });
      handle.pushNamed('finish', { status: 'success' });
    });
    await waitFor(() =>
      expect(screen.getByTestId('log-output')).toBeInTheDocument()
    );

    // Decision uses the largest pane, not the visible one.
    expect(getTailSelect()).toBeInTheDocument();
  });

  it('keeps the line cap when a short finished log ends in a stream error', async () => {
    render(
      <QueryWrapper>
        <TaskLogViewer taskHistoryId="25" taskStatus="SUCCESS" />
      </QueryWrapper>
    );
    await flushPromises();

    const handle = getHandle('25');
    act(() => {
      handle.pushMessage({
        msg: lines(2),
        step: 'setup',
        type: 'stdout',
        offset: 1,
      });
      handle.pushNamed('sep-error', { detail: 'gateway blew up' });
    });
    await waitFor(() =>
      expect(screen.getByText('gateway blew up')).toBeInTheDocument()
    );

    // An aborted stream never proves the log is complete.
    expect(getTailSelect()).toBeInTheDocument();
  });

  it('keeps the line cap visible but disabled for a running task with a short log', async () => {
    render(
      <QueryWrapper>
        <TaskLogViewer taskHistoryId="24" taskStatus="RUNNING" />
      </QueryWrapper>
    );
    await flushPromises();

    const handle = getHandle('24');
    act(() => {
      handle.pushMessage({
        msg: lines(2),
        step: 'setup',
        type: 'stdout',
        offset: 1,
      });
      handle.pushNamed('finish', { status: 'success' });
    });
    await waitFor(() => expect(screen.getByText('Done')).toBeInTheDocument());

    expect(getTailSelect()).toBeInTheDocument();
    expect(getTailSelect()).toHaveAttribute('aria-disabled', 'true');
  });

  it('marks the stderr top tab as unread when stderr arrives while on stdout', async () => {
    render(
      <QueryWrapper>
        <TaskLogViewer taskHistoryId="1" taskStatus="RUNNING" />
      </QueryWrapper>
    );
    await flushPromises();

    const handle = getHandle('1');
    act(() => {
      handle.pushMessage({
        msg: 'err\n',
        step: 'setup',
        type: 'stderr',
        offset: 1,
      });
    });

    await waitFor(() => {
      const stderrTab = screen.getByRole('tab', { name: /stderr/i });
      const dot = stderrTab.querySelector('.MuiBadge-dot');
      expect(dot).toBeTruthy();
      expect(dot?.classList.contains('MuiBadge-invisible')).toBe(false);
    });

    const stderrTab = screen.getByRole('tab', { name: /stderr/i });
    const user = userEvent.setup();
    await user.click(stderrTab);

    // MUI Badge keeps the dot element but toggles an invisible class
    const dotAfter = stderrTab.querySelector('.MuiBadge-dot');
    expect(dotAfter?.classList.contains('MuiBadge-invisible')).toBe(true);
  });

  it('opens the execution events panel from the demoted control in one interaction', async () => {
    render(
      <QueryWrapper>
        <TaskLogViewer taskHistoryId="1" taskStatus="RUNNING" />
      </QueryWrapper>
    );
    await flushPromises();

    const handle = getHandle('1');
    act(() => {
      handle.pushMessage({
        msg: 'out\n',
        step: 'setup',
        type: 'stdout',
        offset: 1,
      });
    });
    await waitFor(() =>
      expect(screen.getByTestId('log-output')).toBeInTheDocument()
    );

    const user = userEvent.setup();
    await user.click(getEventsButton());

    expect(screen.queryByTestId('log-output')).not.toBeInTheDocument();
    expect(screen.getByText(/no execution events yet/i)).toBeInTheDocument();
    expect(getEventsButton()).toHaveAttribute('aria-current', 'true');

    // Clicking again is a stable no-op; the way back is a primary tab.
    await user.click(getEventsButton());
    expect(screen.getByText(/no execution events yet/i)).toBeInTheDocument();
    expect(getEventsButton()).toHaveAttribute('aria-current', 'true');
  });

  it('renders exactly two primary tabs, stdout and stderr', async () => {
    render(
      <QueryWrapper>
        <TaskLogViewer taskHistoryId="30" taskStatus="RUNNING" />
      </QueryWrapper>
    );
    await flushPromises();

    const tabs = within(getPrimaryTabList()).getAllByRole('tab');
    expect(tabs.map((tab) => tab.textContent)).toEqual(['stdout', 'stderr']);
    expect(screen.queryByRole('tab', { name: /execution events/i })).toBeNull();
  });

  it('shows no unread indicator while a running task pushes execution events', async () => {
    const { container } = render(
      <QueryWrapper>
        <TaskLogViewer taskHistoryId="31" taskStatus="RUNNING" />
      </QueryWrapper>
    );
    await flushPromises();

    const eventHandle = getEventHandle('31');
    act(() => {
      eventHandle.pushMessage({
        timestamp: '2026-04-28T10:00:00Z',
        type: 'STEP_STARTED',
        description: 'setup started',
        step: 'setup',
      });
    });
    await flushPromises();

    expect(getEventsButton().querySelector('.MuiBadge-dot')).toBeNull();
    // Nothing anywhere in the console badges while the events view is closed.
    expect(
      container.querySelectorAll('.MuiBadge-dot:not(.MuiBadge-invisible)')
    ).toHaveLength(0);

    // The event really did arrive while the view was closed.
    const user = userEvent.setup();
    await user.click(getEventsButton());
    expect(screen.getByText(/setup started/)).toBeInTheDocument();
  });

  it('leaves the primary tabs unselected for the events view and returns in one click', async () => {
    const consoleError = vi
      .spyOn(console, 'error')
      .mockImplementation(() => {});
    onTestFinished(() => consoleError.mockRestore());

    render(
      <QueryWrapper>
        <TaskLogViewer taskHistoryId="32" taskStatus="RUNNING" />
      </QueryWrapper>
    );
    await flushPromises();

    const handle = getHandle('32');
    act(() => {
      handle.pushMessage({
        msg: 'out\n',
        step: 'setup',
        type: 'stdout',
        offset: 1,
      });
    });
    await waitFor(() =>
      expect(screen.getByTestId('log-output')).toBeInTheDocument()
    );

    const user = userEvent.setup();
    await user.click(getEventsButton());

    const tabs = within(getPrimaryTabList()).getAllByRole('tab');
    expect(
      tabs.every((tab) => tab.getAttribute('aria-selected') === 'false')
    ).toBe(true);
    // MUI warns when Tabs `value` is not one of its children; passing false must not.
    const tabsWarnings = consoleError.mock.calls.filter((call) =>
      call.some((arg) => typeof arg === 'string' && /Tabs/.test(arg))
    );
    expect(tabsWarnings).toEqual([]);

    // An unselected strip must still be reachable by keyboard.
    expect(screen.getByRole('tab', { name: /stdout/i })).toHaveAttribute(
      'tabindex',
      '0'
    );

    await user.click(screen.getByRole('tab', { name: /stdout/i }));
    expect(screen.getByTestId('log-output')).toBeInTheDocument();
    expect(getEventsButton()).not.toHaveAttribute('aria-current');
  });

  it('keeps events search and per-step grouping from the demoted entry point', async () => {
    render(
      <QueryWrapper>
        <TaskLogViewer taskHistoryId="33" taskStatus="RUNNING" />
      </QueryWrapper>
    );
    await flushPromises();

    const handle = getHandle('33');
    act(() => {
      handle.pushMessage({
        msg: 'out\n',
        step: 'log-step',
        type: 'stdout',
        offset: 1,
      });
    });

    const eventHandle = getEventHandle('33');
    act(() => {
      eventHandle.pushMessage({
        timestamp: '2026-04-28T10:00:00Z',
        type: 'STEP_STARTED',
        description: 'setup started',
        step: 'setup',
      });
      eventHandle.pushMessage({
        timestamp: '2026-04-28T10:00:05Z',
        type: 'STEP_FINISHED',
        description: 'build finished',
        step: 'build',
      });
    });
    await flushPromises();

    const user = userEvent.setup();
    await user.click(getEventsButton());

    // The step strip lists the execution-event steps, not the log steps.
    await waitFor(() => {
      const stepTabs = within(screen.getAllByRole('tablist')[1]).getAllByRole(
        'tab'
      );
      expect(stepTabs.map((tab) => tab.textContent)).toEqual([
        'setup',
        'build',
      ]);
    });
    expect(screen.getByText(/setup started/)).toBeInTheDocument();

    // Selecting a step filters the events shown.
    await user.click(screen.getByRole('tab', { name: 'build' }));
    expect(screen.getByText(/build finished/)).toBeInTheDocument();
    expect(screen.queryByText(/setup started/)).toBeNull();

    // The search box still narrows within the selected step.
    await user.type(
      screen.getByPlaceholderText(/search events/i),
      'nothing-matches'
    );
    expect(screen.getByText(/no events match/i)).toBeInTheDocument();
  });

  it('triggers a blob download when the download button is clicked', async () => {
    const createObjectURL = vi.fn(() => 'blob:mock');
    const revokeObjectURL = vi.fn();
    vi.stubGlobal('URL', { ...URL, createObjectURL, revokeObjectURL });
    const clickSpy = vi
      .spyOn(HTMLAnchorElement.prototype, 'click')
      .mockImplementation(() => {});

    render(
      <QueryWrapper>
        <TaskLogViewer taskHistoryId="99" taskStatus="RUNNING" />
      </QueryWrapper>
    );
    await flushPromises();

    const handle = getHandle('99');
    act(() => {
      handle.pushMessage({
        msg: 'payload',
        step: 'run',
        type: 'stdout',
        offset: 1,
      });
    });
    await waitFor(() =>
      expect(screen.getByTestId('log-output')).toBeInTheDocument()
    );

    const user = userEvent.setup();
    await user.click(screen.getByRole('button', { name: /download log/i }));

    expect(createObjectURL).toHaveBeenCalledTimes(1);
    expect(clickSpy).toHaveBeenCalledTimes(1);
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:mock');
  });

  it('renders a status badge when the stream finishes', async () => {
    render(
      <QueryWrapper>
        <TaskLogViewer taskHistoryId="1" taskStatus="RUNNING" />
      </QueryWrapper>
    );
    await flushPromises();

    const handle = getHandle('1');
    act(() => {
      handle.pushNamed('finish', { status: 'success' });
    });
    await waitFor(() => expect(screen.getByText('Done')).toBeInTheDocument());
  });

  it('resets active step and unread state when taskHistoryId changes', async () => {
    const { rerender } = render(
      <QueryWrapper>
        <TaskLogViewer taskHistoryId="1" taskStatus="RUNNING" />
      </QueryWrapper>
    );
    await flushPromises();

    const first = getHandle('1');
    act(() => {
      first.pushMessage({
        msg: 'a\n',
        step: 'alpha',
        type: 'stdout',
        offset: 1,
      });
    });
    await waitFor(() =>
      expect(screen.getByTestId('log-output').textContent).toBe('a\n')
    );

    rerender(
      <QueryWrapper>
        <TaskLogViewer taskHistoryId="2" taskStatus="RUNNING" />
      </QueryWrapper>
    );
    await flushPromises();

    // Previous step alpha no longer exists; empty state until new data arrives
    expect(screen.queryByTestId('log-output')).not.toBeInTheDocument();
    expect(screen.getByText(/no output yet/i)).toBeInTheDocument();

    const second = getHandle('2');
    act(() => {
      second.pushMessage({
        msg: 'b\n',
        step: 'beta',
        type: 'stdout',
        offset: 1,
      });
    });
    await waitFor(() =>
      expect(screen.getByTestId('log-output').textContent).toBe('b\n')
    );
    // New step auto-selected, no unread dots from the prior task
    const stepTab = screen.getByRole('tab', { name: /beta/i });
    expect(within(stepTab).queryByRole('status')).toBeNull();
  });

  it('renders the executor-gone error block for 410', async () => {
    render(
      <QueryWrapper>
        <TaskLogViewer taskHistoryId="1" taskStatus="RUNNING" />
      </QueryWrapper>
    );
    await flushPromises();

    const handle = getHandle('1');
    act(() => {
      handle.pushNamed('sep-error', {
        code: 410,
        detail: { message: 'gone', job_id: 'J-1', executor_name: 'nomad-a' },
      });
    });
    await waitFor(() => expect(screen.getByText('gone')).toBeInTheDocument());
    expect(screen.getByText(/J-1/)).toBeInTheDocument();
    expect(screen.getByText(/nomad-a/)).toBeInTheDocument();
    expect(screen.getByText('Not in executor')).toBeInTheDocument();
  });
});

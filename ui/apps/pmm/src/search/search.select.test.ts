import messenger from 'lib/messenger';
import { performSelection, withDashboardVariables } from './search.select';
import { testItem } from './search.testFixtures';
import type { SearchItem } from './search.types';

const setup = (
  overrides: { pathname?: string; isFrameLoaded?: boolean } = {}
) => {
  const calls: string[] = [];
  const deps = {
    navigate: vi.fn((to: string) => calls.push(`navigate ${to}`)),
    close: vi.fn(() => calls.push('close')),
    closeNavIfNarrow: vi.fn(() => calls.push('closeNav')),
    record: vi.fn((item: SearchItem) => calls.push(`record ${item.id}`)),
    isFrameLoaded: overrides.isFrameLoaded ?? true,
    pathname: overrides.pathname ?? '/help',
  };

  return { deps, calls };
};

const DASHBOARD = testItem('Disk details', {
  id: 'dashboard:node-disk',
  kind: 'dashboard',
  group: 'dashboards',
  to: '/graph/d/node-disk/disk-details',
  carryVariables: true,
});

const nextFrames = (count: number) => {
  for (let i = 0; i < count; i++) {
    vi.advanceTimersToNextFrame();
  }
};

describe('performSelection', () => {
  const originalLocation = window.location;
  let open: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    open = vi.spyOn(window, 'open').mockReturnValue(null);
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.useRealTimers();
    Object.defineProperty(window, 'location', {
      value: originalLocation,
      writable: true,
    });
  });

  it('runs a retry row and keeps the dialog open', async () => {
    const refetch = vi.fn();
    const { deps, calls } = setup();

    await performSelection(
      testItem('Retry', { kind: 'retry', perform: refetch }),
      { newTab: false },
      deps
    );

    expect(refetch).toHaveBeenCalledTimes(1);
    expect(calls).toEqual([]);
  });

  it('closes first and runs an action after the focus restore, leaving the nav alone', async () => {
    vi.useFakeTimers({ toFake: ['requestAnimationFrame'] });
    const perform = vi.fn();
    const { deps, calls } = setup();

    await performSelection(
      testItem('Switch to dark mode', { kind: 'action', perform }),
      { newTab: false },
      deps
    );

    expect(calls).toEqual(['close']);
    nextFrames(1);
    expect(perform).not.toHaveBeenCalled();
    nextFrames(1);
    expect(perform).toHaveBeenCalledTimes(1);
  });

  it('signs out through the real address after its click handler', async () => {
    const assign = vi.fn();
    Object.defineProperty(window, 'location', {
      value: { ...originalLocation, assign },
      writable: true,
    });
    const perform = vi.fn();
    const { deps, calls } = setup();

    await performSelection(
      testItem('Sign out', {
        kind: 'action',
        href: '/graph/logout',
        target: '_self',
        perform,
      }),
      { newTab: false },
      deps
    );

    expect(calls).toEqual(['close', 'closeNav']);
    expect(perform).toHaveBeenCalledTimes(1);
    expect(assign).toHaveBeenCalledWith('/graph/logout');
    expect(open).not.toHaveBeenCalled();
  });

  it('opens external help in a new tab without recording it', async () => {
    const { deps, calls } = setup();

    await performSelection(
      testItem('PMM documentation', {
        kind: 'help',
        href: 'https://per.co.na/pmm_documentation',
        external: true,
      }),
      { newTab: false },
      deps
    );

    expect(open).toHaveBeenCalledWith(
      'https://per.co.na/pmm_documentation',
      '_blank',
      'noopener,noreferrer'
    );
    expect(calls).toEqual(['close', 'closeNav']);
  });

  it('opens a destination in a new tab under the PMM base path and records it', async () => {
    const { deps, calls } = setup();

    await performSelection(DASHBOARD, { newTab: true }, deps);

    expect(open).toHaveBeenCalledWith(
      '/pmm-ui/graph/d/node-disk/disk-details',
      '_blank',
      'noopener'
    );
    expect(calls).toEqual(['close', 'closeNav', 'record dashboard:node-disk']);
  });

  it('carries the open dashboard’s variables to another dashboard', async () => {
    const send = vi
      .spyOn(messenger, 'sendMessageWithResult')
      .mockResolvedValue({ url: '/d/node-disk/disk-details?var-node_name=a' });
    const { deps, calls } = setup({ pathname: '/graph/d/node-cpu/cpu' });

    await performSelection(DASHBOARD, { newTab: false }, deps);

    expect(send).toHaveBeenCalledWith(
      {
        type: 'DASHBOARD_VARIABLES',
        payload: { url: '/d/node-disk/disk-details' },
      },
      300
    );
    expect(calls).toEqual([
      'close',
      'closeNav',
      'record dashboard:node-disk',
      'navigate /graph/d/node-disk/disk-details?var-node_name=a',
    ]);
  });

  it('falls back to the plain link when Grafana does not answer in time', async () => {
    vi.useFakeTimers();
    const { deps } = setup({ pathname: '/graph/d/node-cpu/cpu' });

    const selection = performSelection(DASHBOARD, { newTab: false }, deps);
    await vi.advanceTimersByTimeAsync(299);
    expect(deps.navigate).not.toHaveBeenCalled();
    await vi.advanceTimersByTimeAsync(1);
    await selection;

    expect(deps.navigate).toHaveBeenCalledWith(
      '/graph/d/node-disk/disk-details'
    );
  });

  it.each([
    ['the frame is not loaded', { isFrameLoaded: false }],
    ['it is the open dashboard', { pathname: '/graph/d/node-disk/other-slug' }],
  ])('navigates straight there when %s', async (_, overrides) => {
    const send = vi.spyOn(messenger, 'sendMessageWithResult');
    const { deps } = setup(overrides);

    await performSelection(DASHBOARD, { newTab: false }, deps);

    expect(send).not.toHaveBeenCalled();
    expect(deps.navigate).toHaveBeenCalledWith(
      '/graph/d/node-disk/disk-details'
    );
  });

  it.each([
    ['page', true],
    ['setting', true],
    ['folder', true],
    ['service', true],
    ['node', true],
    ['help', true],
    ['action', false],
    ['link', false],
  ] as const)('navigates to a %s, recording it: %s', async (kind, recorded) => {
    const { deps, calls } = setup();

    await performSelection(
      testItem('Target', { kind, to: '/target' }),
      { newTab: false },
      deps
    );

    expect(calls).toEqual([
      'close',
      'closeNav',
      ...(recorded ? ['record page:Target'] : []),
      'navigate /target',
    ]);
  });
});

describe('withDashboardVariables', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('keeps a returned path that already carries Grafana’s sub path', async () => {
    vi.spyOn(messenger, 'sendMessageWithResult').mockResolvedValue({
      url: '/graph/d/a/b?from=now-1h',
    });

    await expect(withDashboardVariables('/graph/d/a/b')).resolves.toBe(
      '/graph/d/a/b?from=now-1h'
    );
  });

  it('falls back to the link on any failure', async () => {
    vi.spyOn(messenger, 'sendMessageWithResult').mockRejectedValue(undefined);

    await expect(withDashboardVariables('/graph/d/a/b')).resolves.toBe(
      '/graph/d/a/b'
    );
  });
});

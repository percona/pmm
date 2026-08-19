import { act, render, screen } from '@testing-library/react';
import { reloadPage } from 'utils/dom.utils';
import { GetServerVersionResponse } from 'types/server.types';
import { VersionProvider } from './version.provider';
import { useVersion } from './version.hooks';
import {
  AUTO_RELOAD_COOLDOWN_MS,
  canAutoReload,
  recordAutoReload,
} from './version.utils';

const mocks = vi.hoisted(() => ({
  useServerVersion: vi.fn(),
}));

vi.mock('hooks/api/useServerVersion', () => ({
  useServerVersion: mocks.useServerVersion,
}));

vi.mock('utils/dom.utils', async (importOriginal) => ({
  ...(await importOriginal<typeof import('utils/dom.utils')>()),
  reloadPage: vi.fn(),
}));

const build = (fullVersion: string, version = '3.10.0') =>
  ({
    version,
    managed: { version, fullVersion },
  }) satisfies GetServerVersionResponse;

const Consumer = () => {
  const { isOutdated, serverVersion, reload } = useVersion();

  return (
    <>
      <div data-testid="state">{`${isOutdated ? 'outdated' : 'current'}:${serverVersion}`}</div>
      <button data-testid="reload" onClick={reload} />
    </>
  );
};

/** Leaving the document reports it hidden on the way out. */
const unload = () => {
  setVisibility('hidden');
  act(() => {
    document.dispatchEvent(new Event('visibilitychange'));
  });
};

const setVisibility = (state: DocumentVisibilityState) =>
  Object.defineProperty(document, 'visibilityState', {
    value: state,
    configurable: true,
  });

const renderProvider = (data?: GetServerVersionResponse) => {
  mocks.useServerVersion.mockReturnValue({ data });

  return render(
    <VersionProvider>
      <Consumer />
    </VersionProvider>
  );
};

const serveBuild = (
  rerender: (ui: React.ReactElement) => void,
  data: GetServerVersionResponse
) => {
  mocks.useServerVersion.mockReturnValue({ data });
  act(() => {
    rerender(
      <VersionProvider>
        <Consumer />
      </VersionProvider>
    );
  });
};

describe('VersionProvider', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    sessionStorage.clear();
    setVisibility('visible');
  });

  it('takes the first reading as the baseline', () => {
    renderProvider(build('abc123'));

    expect(screen.getByTestId('state')).toHaveTextContent('current:3.10.0');
    expect(reloadPage).not.toHaveBeenCalled();
  });

  it('reports the page as outdated when the build changes', () => {
    const { rerender } = renderProvider(build('abc123'));

    serveBuild(rerender, build('def456', '3.11.0'));

    expect(screen.getByTestId('state')).toHaveTextContent('outdated:3.11.0');
  });

  it('does not reload a visible tab', () => {
    const { rerender } = renderProvider(build('abc123'));

    serveBuild(rerender, build('def456'));

    expect(reloadPage).not.toHaveBeenCalled();
  });

  it('reloads a hidden tab as soon as the build changes', () => {
    const { rerender } = renderProvider(build('abc123'));
    setVisibility('hidden');

    serveBuild(rerender, build('def456'));

    expect(reloadPage).toHaveBeenCalledTimes(1);
  });

  it('reloads once a visible tab is hidden', () => {
    const { rerender } = renderProvider(build('abc123'));
    serveBuild(rerender, build('def456'));
    expect(reloadPage).not.toHaveBeenCalled();

    unload();

    expect(reloadPage).toHaveBeenCalledTimes(1);
  });

  it('leaves the allowance alone when the reload was asked for', () => {
    const { rerender } = renderProvider(build('abc123'));
    serveBuild(rerender, build('def456'));

    act(() => {
      screen.getByTestId('reload').click();
    });
    // the document reports itself hidden as it goes away
    unload();

    expect(reloadPage).toHaveBeenCalledTimes(1);
    expect(canAutoReload(Date.now())).toBe(true);
  });

  it('spends the allowance when it reloads on its own', () => {
    const { rerender } = renderProvider(build('abc123'));
    setVisibility('hidden');

    serveBuild(rerender, build('def456'));

    expect(reloadPage).toHaveBeenCalledTimes(1);
    expect(canAutoReload(Date.now())).toBe(false);
  });

  it('reloads once even when the tab is hidden more than once', () => {
    const { rerender } = renderProvider(build('abc123'));
    setVisibility('hidden');
    serveBuild(rerender, build('def456'));

    unload();
    unload();

    expect(reloadPage).toHaveBeenCalledTimes(1);
  });

  it('does not reload again within the cooldown', () => {
    recordAutoReload(Date.now() - AUTO_RELOAD_COOLDOWN_MS + 1000);
    const { rerender } = renderProvider(build('abc123'));
    setVisibility('hidden');

    serveBuild(rerender, build('def456'));

    expect(reloadPage).not.toHaveBeenCalled();
  });

  it('leaves it to the prompt when the cooldown cannot be stored', () => {
    const setItem = vi
      .spyOn(Storage.prototype, 'setItem')
      .mockImplementation(() => {
        throw new Error('storage disabled');
      });
    const { rerender } = renderProvider(build('abc123'));
    setVisibility('hidden');

    serveBuild(rerender, build('def456'));

    // reloading without a record of it is what turns an HA build flap into a loop
    expect(reloadPage).not.toHaveBeenCalled();
    expect(screen.getByTestId('state')).toHaveTextContent('outdated:3.10.0');

    setItem.mockRestore();
  });

  it('reloads when a stale asset chunk fails to load', () => {
    renderProvider(build('abc123'));

    act(() => {
      window.dispatchEvent(new Event('vite:preloadError'));
    });

    expect(reloadPage).toHaveBeenCalledTimes(1);
  });

  it('does not keep reloading on a chunk that stays unavailable', () => {
    renderProvider(build('abc123'));

    act(() => {
      window.dispatchEvent(new Event('vite:preloadError'));
      window.dispatchEvent(new Event('vite:preloadError'));
    });

    expect(reloadPage).toHaveBeenCalledTimes(1);
  });

  it('stays inert while the build is unknown', () => {
    const { rerender } = renderProvider({ version: '' });
    setVisibility('hidden');

    serveBuild(rerender, build('def456'));

    expect(screen.getByTestId('state')).toHaveTextContent('current:3.10.0');
    expect(reloadPage).not.toHaveBeenCalled();
  });
});

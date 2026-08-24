import {
  afterAll,
  afterEach,
  beforeEach,
  describe,
  expect,
  it,
  vi,
} from 'vitest';
import { cleanup, render, screen, waitFor } from '@testing-library/react';
import { StrictMode, type ReactElement } from 'react';
import { MemoryRouter, useLocation } from 'react-router-dom';
import { useAuth } from './auth.hooks';
import { wrapWithQueryProvider } from 'utils/testUtils';
import { AuthProvider } from './auth.provider';

const mocks = vi.hoisted(() => ({
  rotateToken: vi.fn(),
  useFrontendSettings: vi.fn(),
}));

vi.mock('api/auth', () => ({
  rotateToken: mocks.rotateToken,
}));

vi.mock('hooks/api/useSettings', async (importOriginal) => ({
  ...(await importOriginal()),
  useFrontendSettings: mocks.useFrontendSettings,
}));

const DEEP_LINK = '/graph/d/node-cpu?viewPanel=22';
const RETURN_TO_KEY = 'pmm-ui.auth.returnTo';

let renderedLocations: string[] = [];

/**
 * Records "<url>|<isLoggedIn>" for every render. The invariant that protects the Grafana iframe
 * is that children never render at the stale location *while authenticated* — GrafanaPage
 * snapshots window.location into its iframe src the moment GrafanaProvider reports a session, so
 * a stale render after login would pin the iframe to the wrong dashboard. Renders before the
 * session resolves are harmless: no iframe exists yet.
 */
const Probe = () => {
  const location = useLocation();
  const { isLoggedIn } = useAuth();
  renderedLocations.push(
    `${location.pathname}${location.search}${location.hash}|${isLoggedIn}`
  );
  return <div data-testid="probe" />;
};

const renderProvider = (initialEntry: string, strict = false) => {
  const tree: ReactElement = (
    <MemoryRouter initialEntries={[initialEntry]}>
      {wrapWithQueryProvider(
        <AuthProvider>
          <Probe />
        </AuthProvider>
      )}
    </MemoryRouter>
  );

  return render(strict ? <StrictMode>{tree}</StrictMode> : tree);
};

const storeReturnTo = (path: string) =>
  sessionStorage.setItem(
    RETURN_TO_KEY,
    JSON.stringify({ path, at: Date.now() })
  );

describe('AuthProvider', () => {
  const originalLocation = window.location;
  const replaceMock = vi.fn();

  beforeEach(() => {
    renderedLocations = [];
    replaceMock.mockClear();
    mocks.rotateToken.mockReset();
    mocks.useFrontendSettings.mockReturnValue({
      data: { anonymousEnabled: false },
      isLoading: false,
    });
    // Kept in place for the whole file: a render flushed after a test finishes must still hit the
    // mock, otherwise jsdom logs "Not implemented: navigation".
    Object.defineProperty(window, 'location', {
      value: {
        ...originalLocation,
        pathname: '/pmm-ui/graph/d/node-cpu',
        search: '?viewPanel=22',
        hash: '',
        replace: replaceMock,
      },
      writable: true,
    });
  });

  afterEach(() => {
    // Unmount first: a pending render flushed during cleanup can still write a return-to entry,
    // which would then leak into the next test.
    cleanup();
    sessionStorage.clear();
    localStorage.clear();
  });

  afterAll(() => {
    Object.defineProperty(window, 'location', {
      value: originalLocation,
      writable: true,
    });
  });

  it('remembers the requested URL and leaves for login on 401', async () => {
    mocks.rotateToken.mockRejectedValue({ response: { status: 401 } });

    renderProvider('/graph/d/node-cpu?viewPanel=22');

    await waitFor(() => {
      expect(replaceMock).toHaveBeenCalledWith('/graph/login');
    });
    const stored = sessionStorage.getItem(RETURN_TO_KEY);
    expect(stored).not.toBeNull();
    expect(JSON.parse(stored ?? '{}').path).toBe(DEEP_LINK);
    expect(screen.queryByTestId('probe')).not.toBeInTheDocument();
  });

  it('restores the remembered URL before rendering children', async () => {
    mocks.rotateToken.mockResolvedValue({ token: 'ok' });
    storeReturnTo(DEEP_LINK);

    renderProvider('/graph');

    await screen.findByTestId('probe');
    expect(renderedLocations).not.toContain('/graph|true');
    expect(renderedLocations).toContain(`${DEEP_LINK}|true`);
    expect(sessionStorage.getItem(RETURN_TO_KEY)).toBeNull();
  });

  it('does not navigate when the remembered URL is already the current one', async () => {
    mocks.rotateToken.mockResolvedValue({ token: 'ok' });
    storeReturnTo(DEEP_LINK);

    renderProvider(DEEP_LINK);

    await screen.findByTestId('probe');
    expect(renderedLocations).toContain(`${DEEP_LINK}|true`);
    expect(sessionStorage.getItem(RETURN_TO_KEY)).toBeNull();
  });

  it('renders children in place when nothing was remembered', async () => {
    mocks.rotateToken.mockResolvedValue({ token: 'ok' });

    renderProvider('/graph');

    await screen.findByTestId('probe');
    expect(renderedLocations).toContain('/graph|true');
    expect(replaceMock).not.toHaveBeenCalled();
  });

  it('neither redirects nor remembers anything when anonymous access is on', async () => {
    mocks.rotateToken.mockRejectedValue({ response: { status: 401 } });
    mocks.useFrontendSettings.mockReturnValue({
      data: { anonymousEnabled: true },
      isLoading: false,
    });

    renderProvider('/graph/d/node-cpu?viewPanel=22');

    await screen.findByTestId('probe');
    expect(replaceMock).not.toHaveBeenCalled();
    expect(sessionStorage.getItem(RETURN_TO_KEY)).toBeNull();
  });

  it('restores once under StrictMode', async () => {
    mocks.rotateToken.mockResolvedValue({ token: 'ok' });
    storeReturnTo(DEEP_LINK);

    renderProvider('/graph', true);

    await screen.findByTestId('probe');
    expect(renderedLocations).not.toContain('/graph|true');
    expect(renderedLocations[renderedLocations.length - 1]).toBe(
      `${DEEP_LINK}|true`
    );
    expect(sessionStorage.getItem(RETURN_TO_KEY)).toBeNull();
  });
});

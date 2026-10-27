import { act, render, screen, waitFor } from '@testing-library/react';
import { QueryClient } from '@tanstack/react-query';
import { api } from 'api/api';
import { getServerVersion } from 'api/server';
import { SERVER_VERSION_QUERY_KEY } from 'hooks/api/useServerVersion';
import { reloadPage } from 'utils/dom.utils';
import { wrapWithQueryProvider } from 'utils/testUtils';
import { VersionProvider } from './version.provider';
import { useVersion } from './version.hooks';
import { getServerBuildId } from './version.utils';

vi.mock('utils/dom.utils', async (importOriginal) => ({
  ...(await importOriginal<typeof import('utils/dom.utils')>()),
  reloadPage: vi.fn(),
}));

// Captured verbatim from GET /v1/server/version on percona/pmm-server:3.7.1 and
// on a 3.8.0 server: snake_case, as the REST gateway puts it on the wire.
const WIRE_3_7_1 = {
  version: '3.7.1',
  server: {
    version: '3.7.1',
    full_version: '3.7.1',
    timestamp: '2026-04-24T10:50:20Z',
  },
  managed: {
    version: '3.7.1',
    full_version: '7ed36f3434a50bbcd64ef97e778ec81e63f7a5a9',
    timestamp: '2026-04-24T10:50:20Z',
  },
  distribution_method: 'DISTRIBUTION_METHOD_DOCKER',
};

const WIRE_3_8_0 = {
  version: '3.8.0',
  server: {
    version: '3.8.0',
    full_version: '3.8.0',
    timestamp: '2026-05-15T10:11:22Z',
  },
  managed: {
    version: '3.8.0',
    full_version: 'fa62c8c694a54d1023db52204dd1b9d8748b26e9',
    timestamp: '2026-05-15T10:11:22Z',
  },
  distribution_method: 'DISTRIBUTION_METHOD_DOCKER',
};

let served: object = WIRE_3_7_1;

const Consumer = () => {
  const { isOutdated, serverVersion } = useVersion();

  return (
    <div data-testid="state">{`${isOutdated ? 'outdated' : 'current'}:${serverVersion}`}</div>
  );
};

const setVisibility = (state: DocumentVisibilityState) =>
  Object.defineProperty(document, 'visibilityState', {
    value: state,
    configurable: true,
  });

const startTab = () => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  render(
    wrapWithQueryProvider(
      <VersionProvider>
        <Consumer />
      </VersionProvider>,
      client
    )
  );

  return client;
};

/** The server the tab talks to is replaced by the next version. */
const upgradeServer = async (client: QueryClient) => {
  served = WIRE_3_8_0;

  await act(async () => {
    await client.invalidateQueries({ queryKey: SERVER_VERSION_QUERY_KEY });
  });
};

describe('server version watcher, from the wire', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    sessionStorage.clear();
    setVisibility('visible');
    served = WIRE_3_7_1;
    api.defaults.adapter = async (config) => ({
      data: served,
      status: 200,
      statusText: 'OK',
      headers: {},
      config,
    });
  });

  it('finds the commit under the name the API uses for it', async () => {
    const data = await getServerVersion();

    // `full_version` on the wire, `fullVersion` by the time it reaches the types
    expect(data.managed?.fullVersion).toBe(
      '7ed36f3434a50bbcd64ef97e778ec81e63f7a5a9'
    );
    expect(getServerBuildId(data)).toBe(
      '7ed36f3434a50bbcd64ef97e778ec81e63f7a5a9'
    );
  });

  it('holds still while the server keeps answering with the same build', async () => {
    const client = startTab();

    await waitFor(() =>
      expect(screen.getByTestId('state')).toHaveTextContent('current:3.7.1')
    );

    await act(async () => {
      await client.invalidateQueries({ queryKey: SERVER_VERSION_QUERY_KEY });
      await client.invalidateQueries({ queryKey: SERVER_VERSION_QUERY_KEY });
    });

    expect(screen.getByTestId('state')).toHaveTextContent('current:3.7.1');
    expect(reloadPage).not.toHaveBeenCalled();
  });

  it('notices the upgrade and names the new version', async () => {
    const client = startTab();
    await waitFor(() =>
      expect(screen.getByTestId('state')).toHaveTextContent('current:3.7.1')
    );

    await upgradeServer(client);

    await waitFor(() =>
      expect(screen.getByTestId('state')).toHaveTextContent('outdated:3.8.0')
    );
    expect(reloadPage).not.toHaveBeenCalled();
  });

  it('reloads a backgrounded tab once the upgrade lands', async () => {
    const client = startTab();
    await waitFor(() =>
      expect(screen.getByTestId('state')).toHaveTextContent('current:3.7.1')
    );
    setVisibility('hidden');

    await upgradeServer(client);

    await waitFor(() => expect(reloadPage).toHaveBeenCalledTimes(1));
  });
});

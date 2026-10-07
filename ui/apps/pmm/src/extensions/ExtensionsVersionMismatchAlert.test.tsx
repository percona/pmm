import { act, render, screen, waitFor } from '@testing-library/react';
import type { ReactElement } from 'react';
import { apiClient, postSessionExchange } from '@pmm-extensions/api';
import { TestWrapper } from 'utils/testWrapper';
import {
  wrapWithQueryProvider,
  wrapWithSettings,
  wrapWithVersion,
} from 'utils/testUtils';
import { initExtensionsAuth } from './bootstrap';
import { ExtensionsPage } from './ExtensionsPage';
import { ExtensionsVersionMismatchAlert } from './ExtensionsVersionMismatchAlert';
import { Messages } from './ExtensionsVersionMismatchAlert.messages';
import { resetExtensionsAuthStore } from './extensionsTokenStore';

// Keep the real useAppInfo / apiClient; only the session exchange is stubbed so
// AuthGate can be driven without a live side-car.
vi.mock('@pmm-extensions/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@pmm-extensions/api')>()),
  postSessionExchange: vi.fn(),
}));

const exchange = vi.mocked(postSessionExchange);

const bearer = (accessToken = 'bearer-1') => ({
  access_token: accessToken,
  expires_in: 300,
});

const stubAppInfoResponse = (version: string | null) =>
  vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
    if (url === '/extensions/app-info/') {
      return {
        data: { footer_text: '', version },
      };
    }
    throw new Error(`unexpected GET ${url}`);
  });

beforeEach(() => {
  exchange.mockReset();
  resetExtensionsAuthStore();
  initExtensionsAuth();
});

afterEach(() => {
  resetExtensionsAuthStore();
  vi.restoreAllMocks();
});

describe('ExtensionsVersionMismatchAlert (real useAppInfo)', () => {
  it('warns from the app-info version returned by apiClient.get', async () => {
    stubAppInfoResponse('3.11.0');

    render(
      wrapWithQueryProvider(
        wrapWithVersion(<ExtensionsVersionMismatchAlert />, {
          serverVersion: '3.10.0',
        })
      )
    );

    expect(
      await screen.findByTestId('extensions-version-mismatch')
    ).toHaveTextContent(Messages.mismatch('3.10.0', '3.11.0'));
    expect(apiClient.get).toHaveBeenCalledWith('/extensions/app-info/');
  });
});

describe('ExtensionsPage version warning after AuthGate', () => {
  const renderPage = () =>
    render(
      <ExtensionsPage>
        <div data-testid="extensions-plugin" />
      </ExtensionsPage>,
      {
        wrapper: ({ children }) => (
          <TestWrapper>
            {wrapWithQueryProvider(
              wrapWithVersion(
                wrapWithSettings(children as ReactElement, {
                  settings: { extensionsEnabled: true },
                }),
                { serverVersion: '3.10.0' }
              )
            )}
          </TestWrapper>
        ),
      }
    );

  it('fetches app-info only after the auth gate is ready', async () => {
    let resolveExchange: (value: ReturnType<typeof bearer>) => void = () => {};
    exchange.mockReturnValue(
      new Promise((resolve) => {
        resolveExchange = resolve;
      })
    );
    const get = stubAppInfoResponse('3.11.0');

    renderPage();

    expect(screen.queryByTestId('extensions-plugin')).not.toBeInTheDocument();
    expect(
      screen.queryByTestId('extensions-version-mismatch')
    ).not.toBeInTheDocument();
    expect(get).not.toHaveBeenCalled();

    await act(async () => {
      resolveExchange(bearer());
    });

    expect(await screen.findByTestId('extensions-plugin')).toBeInTheDocument();
    expect(
      await screen.findByTestId('extensions-version-mismatch')
    ).toHaveTextContent(Messages.mismatch('3.10.0', '3.11.0'));
    await waitFor(() => {
      expect(get).toHaveBeenCalledWith('/extensions/app-info/');
    });
  });
});

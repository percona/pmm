import { render, screen } from '@testing-library/react';
import type { ReactElement } from 'react';
import { useAppInfo } from '@pmm-extensions/api';
import { SettingsContext } from 'contexts/settings';
import type { VersionContextProps } from 'contexts/version';
import { TestWrapper } from 'utils/testWrapper';
import {
  measurePageSurface,
  measureSurface,
  wrapWithSettings,
  wrapWithVersion,
} from 'utils/testUtils';
import { TEST_USER_ADMIN, TEST_USER_VIEWER } from 'utils/testStubs';
import type { User } from 'types/user.types';
import { ExtensionsPage } from './ExtensionsPage';
import { Messages as VersionMismatchMessages } from './ExtensionsVersionMismatchAlert.messages';

// The gate mints a side-car bearer on mount; this suite is about who reaches it, so
// hold it open and let the page render its children.
vi.mock('./ExtensionsAuthGate', () => ({
  ExtensionsAuthGate: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
}));

// Settings-gating cases never needed app-info; mocking useAppInfo keeps them
// free of a QueryClient and means they no longer hit the real hook. Version
// mismatch cases stub the return value per assertion instead.
vi.mock('@pmm-extensions/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@pmm-extensions/api')>();
  return {
    ...actual,
    useAppInfo: vi.fn(),
  };
});

const appInfo = vi.mocked(useAppInfo);

const pendingAppInfo = {
  data: undefined,
  isSuccess: false,
  isError: false,
  isPending: true,
} as ReturnType<typeof useAppInfo>;

const stubAppInfo = (
  partial: Partial<{
    data: { footer_text: string; version?: string | null };
    isSuccess: boolean;
    isError: boolean;
    isPending: boolean;
  }>
) => {
  appInfo.mockReturnValue({
    data: undefined,
    isSuccess: false,
    isError: false,
    isPending: false,
    ...partial,
  } as ReturnType<typeof useAppInfo>);
};

beforeEach(() => {
  appInfo.mockReturnValue(pendingAppInfo);
});

const renderExtensionsPage = ({
  user = TEST_USER_ADMIN,
  isLoading = false,
  settings = { extensionsEnabled: true },
  version,
}: {
  user?: User;
  isLoading?: boolean;
  settings?: { extensionsEnabled?: boolean };
  version?: Partial<VersionContextProps>;
} = {}) =>
  render(
    <ExtensionsPage>
      <div data-testid="extensions-plugin" />
    </ExtensionsPage>,
    {
      wrapper: ({ children }) => {
        let tree = wrapWithSettings(children as ReactElement, {
          isLoading,
          settings,
        });
        if (version !== undefined) {
          tree = wrapWithVersion(tree, version);
        }
        return (
          <TestWrapper userContext={{ isLoading: false, user }}>
            {tree}
          </TestWrapper>
        );
      },
    }
  );

describe('ExtensionsPage', () => {
  it('renders the plugin for an administrator', () => {
    renderExtensionsPage();

    expect(screen.getByTestId('extensions-plugin')).toBeInTheDocument();
  });

  it('renders the plugin for a viewer rather than an unauthorized card', () => {
    // The side-car serves its reads to any authenticated session and holds every unsafe
    // method to administrators, so the route carries no role restriction and
    // the write controls are withheld per control instead (PMM-15358).
    renderExtensionsPage({ user: TEST_USER_VIEWER });

    expect(screen.getByTestId('extensions-plugin')).toBeInTheDocument();
  });

  it('renders an unavailable message when PMM Extensions is disabled', () => {
    renderExtensionsPage({ settings: { extensionsEnabled: false } });

    expect(
      screen.getByText(
        'This feature is not enabled. Contact your administrator.'
      )
    ).toBeInTheDocument();
    expect(screen.queryByTestId('extensions-plugin')).not.toBeInTheDocument();
  });

  it('waits for settings instead of flashing not-enabled while they load', () => {
    renderExtensionsPage({
      isLoading: true,
      settings: { extensionsEnabled: true },
    });

    expect(
      screen.getByTestId('extensions-settings-loading')
    ).toBeInTheDocument();
    expect(
      screen.queryByText(
        'This feature is not enabled. Contact your administrator.'
      )
    ).not.toBeInTheDocument();
    expect(screen.queryByTestId('extensions-plugin')).not.toBeInTheDocument();
  });

  it('waits while settings are still null after the default context', () => {
    render(
      <ExtensionsPage>
        <div data-testid="extensions-plugin" />
      </ExtensionsPage>,
      {
        wrapper: ({ children }) => (
          <TestWrapper
            userContext={{ isLoading: false, user: TEST_USER_ADMIN }}
          >
            <SettingsContext.Provider
              value={{ isLoading: false, settings: null }}
            >
              {children}
            </SettingsContext.Provider>
          </TestWrapper>
        ),
      }
    );

    expect(
      screen.getByTestId('extensions-settings-loading')
    ).toBeInTheDocument();
    expect(
      screen.queryByText(
        'This feature is not enabled. Contact your administrator.'
      )
    ).not.toBeInTheDocument();
  });

  it('renders every state on the paper surface Settings uses', () => {
    const stage = measurePageSurface('default');
    const paper = measurePageSurface('paper');

    // Guards the rest of the assertions: they only mean anything while the two
    // surfaces actually differ.
    expect(paper).not.toBe(stage);

    const loading = measureSurface(() =>
      renderExtensionsPage({
        isLoading: true,
        settings: { extensionsEnabled: true },
      })
    );
    const notEnabled = measureSurface(() =>
      renderExtensionsPage({ settings: { extensionsEnabled: false } })
    );
    const loaded = measureSurface(() =>
      renderExtensionsPage({ settings: { extensionsEnabled: true } })
    );

    expect(loading).toBe(paper);
    expect(notEnabled).toBe(paper);
    expect(loaded).toBe(paper);
  });

  describe('version mismatch warning (PMM-15671)', () => {
    it('hides the warning when releases are equal', () => {
      stubAppInfo({
        data: { footer_text: '', version: '3.10.0' },
        isSuccess: true,
      });

      renderExtensionsPage({ version: { serverVersion: '3.10.0' } });

      expect(
        screen.queryByTestId('extensions-version-mismatch')
      ).not.toBeInTheDocument();
      expect(screen.getByTestId('extensions-plugin')).toBeInTheDocument();
    });

    it('warns when releases differ and keeps the page usable', () => {
      stubAppInfo({
        data: { footer_text: '', version: '3.11.0' },
        isSuccess: true,
      });

      renderExtensionsPage({ version: { serverVersion: '3.10.0' } });

      expect(
        screen.getByTestId('extensions-version-mismatch')
      ).toHaveTextContent(VersionMismatchMessages.mismatch('3.10.0', '3.11.0'));
      expect(screen.getByTestId('extensions-plugin')).toBeInTheDocument();
    });

    it('ignores a leading v and a development suffix', () => {
      stubAppInfo({
        data: { footer_text: '', version: 'v3.10.0.dev0' },
        isSuccess: true,
      });

      renderExtensionsPage({ version: { serverVersion: '3.10.0' } });

      expect(
        screen.queryByTestId('extensions-version-mismatch')
      ).not.toBeInTheDocument();
      expect(screen.getByTestId('extensions-plugin')).toBeInTheDocument();
    });

    it('warns when PMM Extensions answers without a version', () => {
      stubAppInfo({
        data: { footer_text: 'PMM Extensions' },
        isSuccess: true,
      });

      renderExtensionsPage({ version: { serverVersion: '3.10.0' } });

      expect(
        screen.getByTestId('extensions-version-mismatch')
      ).toHaveTextContent(VersionMismatchMessages.undetermined);
      expect(screen.getByTestId('extensions-plugin')).toBeInTheDocument();
    });

    it('hides the warning while app-info is still loading', () => {
      stubAppInfo({ isPending: true });

      renderExtensionsPage({ version: { serverVersion: '3.10.0' } });

      expect(
        screen.queryByTestId('extensions-version-mismatch')
      ).not.toBeInTheDocument();
      expect(screen.getByTestId('extensions-plugin')).toBeInTheDocument();
    });

    it('hides the warning when app-info fails', () => {
      stubAppInfo({ isError: true });

      renderExtensionsPage({ version: { serverVersion: '3.10.0' } });

      expect(
        screen.queryByTestId('extensions-version-mismatch')
      ).not.toBeInTheDocument();
      expect(screen.getByTestId('extensions-plugin')).toBeInTheDocument();
    });

    it('hides the warning while the server version is still unknown', () => {
      stubAppInfo({
        data: { footer_text: '', version: '3.11.0' },
        isSuccess: true,
      });

      renderExtensionsPage({ version: { serverVersion: '' } });

      expect(
        screen.queryByTestId('extensions-version-mismatch')
      ).not.toBeInTheDocument();
      expect(screen.getByTestId('extensions-plugin')).toBeInTheDocument();
    });

    it('hides the warning when the server version is unparsable', () => {
      stubAppInfo({
        data: { footer_text: '', version: '3.11.0' },
        isSuccess: true,
      });

      renderExtensionsPage({ version: { serverVersion: 'dev' } });

      expect(
        screen.queryByTestId('extensions-version-mismatch')
      ).not.toBeInTheDocument();
      expect(screen.getByTestId('extensions-plugin')).toBeInTheDocument();
    });
  });
});

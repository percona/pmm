import { render, screen } from '@testing-library/react';
import { ReactElement } from 'react';
import { SettingsContext } from 'contexts/settings';
import { TestWrapper } from 'utils/testWrapper';
import {
  measurePageSurface,
  measureSurface,
  wrapWithSettings,
} from 'utils/testUtils';
import { TEST_USER_ADMIN, TEST_USER_VIEWER } from 'utils/testStubs';
import { User } from 'types/user.types';
import { ExtensionsPage } from './ExtensionsPage';

// The gate mints a side-car bearer on mount; this suite is about who reaches it, so
// hold it open and let the page render its children.
vi.mock('./ExtensionsAuthGate', () => ({
  ExtensionsAuthGate: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
}));

const renderExtensionsPage = ({
  user = TEST_USER_ADMIN,
  isLoading = false,
  settings = { extensionsEnabled: true },
}: {
  user?: User;
  isLoading?: boolean;
  settings?: { extensionsEnabled?: boolean };
} = {}) =>
  render(
    <ExtensionsPage>
      <div data-testid="extensions-plugin" />
    </ExtensionsPage>,
    {
      wrapper: ({ children }) => (
        <TestWrapper userContext={{ isLoading: false, user }}>
          {wrapWithSettings(children as ReactElement, { isLoading, settings })}
        </TestWrapper>
      ),
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
});

import { render, screen } from '@testing-library/react';
import { ReactElement } from 'react';
import { SettingsContext } from 'contexts/settings';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithSettings } from 'utils/testUtils';
import { TEST_USER_ADMIN, TEST_USER_VIEWER } from 'utils/testStubs';
import { User } from 'types/user.types';
import { SepPage } from './SepPage';

// The gate mints a SEP bearer on mount; this suite is about who reaches it, so
// hold it open and let the page render its children.
vi.mock('./SepAuthGate', () => ({
  SepAuthGate: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

const renderPage = (
  user: User,
  settings: { sepEnabled?: boolean } = { sepEnabled: true }
) =>
  render(
    <SepPage>
      <div data-testid="sep-plugin" />
    </SepPage>,
    {
      wrapper: ({ children }) => (
        <TestWrapper userContext={{ isLoading: false, user }}>
          {wrapWithSettings(children as ReactElement, { settings })}
        </TestWrapper>
      ),
    }
  );

describe('SepPage', () => {
  it('renders the plugin for an administrator', () => {
    renderPage(TEST_USER_ADMIN);

    expect(screen.getByTestId('sep-plugin')).toBeInTheDocument();
  });

  it('renders the plugin for a viewer rather than an unauthorized card', () => {
    // SEP serves its reads to any authenticated session and holds every unsafe
    // method to administrators, so the route carries no role restriction and
    // the write controls are withheld per control instead (PMM-15358).
    renderPage(TEST_USER_VIEWER);

    expect(screen.getByTestId('sep-plugin')).toBeInTheDocument();
  });

  it('renders an unavailable message when SEP is disabled', () => {
    renderPage(TEST_USER_ADMIN, { sepEnabled: false });

    expect(
      screen.getByText(
        'This feature is not enabled. Contact your administrator.'
      )
    ).toBeInTheDocument();
    expect(screen.queryByTestId('sep-plugin')).not.toBeInTheDocument();
  });

  it('waits for settings instead of flashing not-enabled while they load', () => {
    render(
      <SepPage>
        <div data-testid="sep-plugin" />
      </SepPage>,
      {
        wrapper: ({ children }) => (
          <TestWrapper
            userContext={{ isLoading: false, user: TEST_USER_ADMIN }}
          >
            {wrapWithSettings(children as ReactElement, {
              isLoading: true,
              settings: { sepEnabled: true },
            })}
          </TestWrapper>
        ),
      }
    );

    expect(screen.getByTestId('sep-settings-loading')).toBeInTheDocument();
    expect(
      screen.queryByText(
        'This feature is not enabled. Contact your administrator.'
      )
    ).not.toBeInTheDocument();
    expect(screen.queryByTestId('sep-plugin')).not.toBeInTheDocument();
  });

  it('waits while settings are still null after the default context', () => {
    render(
      <SepPage>
        <div data-testid="sep-plugin" />
      </SepPage>,
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

    expect(screen.getByTestId('sep-settings-loading')).toBeInTheDocument();
    expect(
      screen.queryByText(
        'This feature is not enabled. Contact your administrator.'
      )
    ).not.toBeInTheDocument();
  });
});

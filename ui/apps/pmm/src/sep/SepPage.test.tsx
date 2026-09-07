import { render, screen } from '@testing-library/react';
import { ReactElement } from 'react';
import { SettingsContext } from 'contexts/settings';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithSettings } from 'utils/testUtils';
import { TEST_USER_ADMIN, TEST_USER_VIEWER } from 'utils/testStubs';
import { User } from 'types/user.types';
import { Page } from 'components/page';
import { SepPage } from './SepPage';

// The gate mints a SEP bearer on mount; this suite is about who reaches it, so
// hold it open and let the page render its children.
vi.mock('./SepAuthGate', () => ({
  SepAuthGate: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

const renderSepPage = ({
  user = TEST_USER_ADMIN,
  isLoading = false,
  settings = { sepEnabled: true },
}: {
  user?: User;
  isLoading?: boolean;
  settings?: { sepEnabled?: boolean };
} = {}) =>
  render(
    <SepPage>
      <div data-testid="sep-plugin" />
    </SepPage>,
    {
      wrapper: ({ children }) => (
        <TestWrapper userContext={{ isLoading: false, user }}>
          {wrapWithSettings(children as ReactElement, { isLoading, settings })}
        </TestWrapper>
      ),
    }
  );

// `Page` paints its surface with a <GlobalStyles> rule on html/body, so the
// resulting background is read back off the document rather than off a node.
const bodyBackground = () => getComputedStyle(document.body).backgroundColor;

// A bare `Page` on each surface, to name the two colours without hardcoding
// them: `surface="paper"` is what Settings passes.
const renderPlainPage = (surface: 'default' | 'paper') =>
  render(
    <Page maxWidth="full" surface={surface}>
      <div />
    </Page>,
    {
      wrapper: ({ children }) => <TestWrapper>{children}</TestWrapper>,
    }
  );

const measureSurface = (renderState: () => { unmount: () => void }) => {
  const { unmount } = renderState();
  const background = bodyBackground();
  unmount();

  return background;
};

describe('SepPage', () => {
  it('renders the plugin for an administrator', () => {
    renderSepPage();

    expect(screen.getByTestId('sep-plugin')).toBeInTheDocument();
  });

  it('renders the plugin for a viewer rather than an unauthorized card', () => {
    // SEP serves its reads to any authenticated session and holds every unsafe
    // method to administrators, so the route carries no role restriction and
    // the write controls are withheld per control instead (PMM-15358).
    renderSepPage({ user: TEST_USER_VIEWER });

    expect(screen.getByTestId('sep-plugin')).toBeInTheDocument();
  });

  it('renders an unavailable message when SEP is disabled', () => {
    renderSepPage({ settings: { sepEnabled: false } });

    expect(
      screen.getByText(
        'This feature is not enabled. Contact your administrator.'
      )
    ).toBeInTheDocument();
    expect(screen.queryByTestId('sep-plugin')).not.toBeInTheDocument();
  });

  it('waits for settings instead of flashing not-enabled while they load', () => {
    renderSepPage({ isLoading: true, settings: { sepEnabled: true } });

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

  it('renders every state on the paper surface Settings uses', () => {
    const stage = measureSurface(() => renderPlainPage('default'));
    const paper = measureSurface(() => renderPlainPage('paper'));

    // Guards the rest of the assertions: they only mean anything while the two
    // surfaces actually differ.
    expect(paper).not.toBe(stage);

    const loading = measureSurface(() =>
      renderSepPage({ isLoading: true, settings: { sepEnabled: true } })
    );
    const notEnabled = measureSurface(() =>
      renderSepPage({ settings: { sepEnabled: false } })
    );
    const loaded = measureSurface(() =>
      renderSepPage({ settings: { sepEnabled: true } })
    );

    expect(loading).toBe(paper);
    expect(notEnabled).toBe(paper);
    expect(loaded).toBe(paper);
  });
});

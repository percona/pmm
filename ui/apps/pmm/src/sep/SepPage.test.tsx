import { render, screen } from '@testing-library/react';
import { TestWrapper } from 'utils/testWrapper';
import { TEST_USER_ADMIN, TEST_USER_VIEWER } from 'utils/testStubs';
import { User } from 'types/user.types';
import { SepPage } from './SepPage';

// The gate mints a SEP bearer on mount; this suite is about who reaches it, so
// hold it open and let the page render its children.
vi.mock('./SepAuthGate', () => ({
  SepAuthGate: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

const renderPage = (user: User) =>
  render(
    <SepPage>
      <div data-testid="sep-plugin" />
    </SepPage>,
    {
      wrapper: ({ children }) => (
        <TestWrapper userContext={{ isLoading: false, user }}>
          {children}
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
});

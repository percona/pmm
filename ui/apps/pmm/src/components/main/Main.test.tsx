import { screen, render } from '@testing-library/react';
import { AuthContext } from 'contexts/auth/auth.context';
import { Main } from './Main';
import { TestWrapper } from 'utils/testWrapper';

describe('Main', () => {
  it('shows loading', () => {
    render(
      <TestWrapper>
        <AuthContext.Provider value={{ isLoading: true }}>
          <Main />
        </AuthContext.Provider>
      </TestWrapper>
    );

    expect(screen.queryByTestId('pmm-loading-indicator')).not.toBeNull();
  });

  it("doesn't show loading", () => {
    render(
      <TestWrapper>
        <AuthContext.Provider value={{ isLoading: false }}>
          <Main />
        </AuthContext.Provider>
      </TestWrapper>
    );

    expect(screen.queryByTestId('pmm-loading-indicator')).toBeNull();
  });
});

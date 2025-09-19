import { screen, render } from '@testing-library/react';
import { Page } from './Page';
import { TestWrapper } from 'utils/testWrapper';
import { UserContext } from 'contexts/user';
import { Messages } from './Page.messages';
import { TEST_USER_ADMIN } from 'utils/testStubs';

describe('Page', () => {
  it('it shows page content when authorized', () => {
    render(
      <TestWrapper>
        <UserContext.Provider
          value={{
            isLoading: false,
            user: TEST_USER_ADMIN,
          }}
        >
          <Page>
            <div>Authorized</div>
          </Page>
        </UserContext.Provider>
      </TestWrapper>
    );

    expect(screen.queryByText('Page Content')).toBeDefined();
  });

  it('it shows no access page when unauthorized', () => {
    render(
      <TestWrapper>
        <UserContext.Provider
          value={{
            isLoading: false,
            user: { ...TEST_USER_ADMIN, isAuthorized: false },
          }}
        >
          <Page>
            <div>Page Content</div>
          </Page>
        </UserContext.Provider>
      </TestWrapper>
    );

    expect(screen.queryByText('Page Content')).toBeNull();
    expect(screen.queryByText(Messages.noAcccess)).toBeDefined();
  });
});

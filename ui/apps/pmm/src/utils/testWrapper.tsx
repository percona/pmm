import { type AuthContextProps, AuthContext } from 'contexts/auth';
import { type UserContextProps, UserContext } from 'contexts/user';
import type { FC, PropsWithChildren } from 'react';
import { type MemoryRouterProps, MemoryRouter } from 'react-router-dom';
import { pmmThemeOptions, ThemeContextProvider } from '@percona/peak-ui';
import { TEST_USER_ADMIN } from './testStubs';

interface TestWrapperProps extends PropsWithChildren {
  authContext?: AuthContextProps;
  userContext?: UserContextProps;
  routerProps?: MemoryRouterProps;
}

export const TestWrapper: FC<TestWrapperProps> = ({
  children,
  authContext = { isLoading: false, isLoggedIn: true },
  userContext = {
    isLoading: false,
    user: TEST_USER_ADMIN,
  },
  routerProps = {},
}) => (
  <AuthContext.Provider value={authContext}>
    <UserContext.Provider value={userContext}>
      <MemoryRouter {...routerProps}>
        <ThemeContextProvider themeOptions={pmmThemeOptions}>
          {children}
        </ThemeContextProvider>
      </MemoryRouter>
    </UserContext.Provider>
  </AuthContext.Provider>
);

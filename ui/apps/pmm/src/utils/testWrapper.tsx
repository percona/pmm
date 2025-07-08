import { ThemeContextProvider } from '@percona/design';
import { AuthContext, AuthContextProps } from 'contexts/auth';
import { UserContext, UserContextProps } from 'contexts/user';
import { FC, PropsWithChildren } from 'react';
import { MemoryRouter, MemoryRouterProps } from 'react-router-dom';
import pmmThemeOptions from 'themes/PmmTheme';
import { OrgRole } from 'types/user.types';

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
    user: {
      id: 1,
      isAuthorized: true,
      isPMMAdmin: true,
      orgRole: OrgRole.Admin,
    },
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

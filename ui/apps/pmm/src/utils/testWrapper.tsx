import { AuthContext, AuthContextProps } from 'contexts/auth';
import { UserContext, UserContextProps } from 'contexts/user';
import { FC, PropsWithChildren } from 'react';
import { MemoryRouter, MemoryRouterProps } from 'react-router-dom';
import { pmmThemeOptions, ThemeContextProvider } from '@percona/peak-ui';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFnsV3';
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
          <LocalizationProvider dateAdapter={AdapterDateFns}>
            {children}
          </LocalizationProvider>
        </ThemeContextProvider>
      </MemoryRouter>
    </UserContext.Provider>
  </AuthContext.Provider>
);

import { ThemeContextProvider } from '@percona/design';
import { AuthContext } from 'contexts/auth';
import { UserContext } from 'contexts/user';
import { FC, PropsWithChildren } from 'react';
import { MemoryRouter } from 'react-router-dom';
import pmmThemeOptions from 'themes/PmmTheme';
import { OrgRole } from 'types/user.types';

export const TestWrapper: FC<PropsWithChildren> = ({ children }) => (
  <AuthContext.Provider value={{ isLoading: false }}>
    <UserContext.Provider
      value={{
        isLoading: false,
        user: {
          id: 1,
          isAuthorized: true,
          isPMMAdmin: true,
          orgRole: OrgRole.Admin,
        },
      }}
    >
      <MemoryRouter>
        <ThemeContextProvider themeOptions={pmmThemeOptions}>
          {children}
        </ThemeContextProvider>
      </MemoryRouter>
    </UserContext.Provider>
  </AuthContext.Provider>
);

import { ThemeContextProvider } from '@percona/design';
import { AuthContext } from 'contexts/auth';
import { FC, PropsWithChildren } from 'react';
import { MemoryRouter } from 'react-router-dom';
import pmmThemeOptions from 'themes/PmmTheme';

export const TestWrapper: FC<PropsWithChildren> = ({ children }) => (
  <AuthContext.Provider value={{ isLoading: false }}>
    <MemoryRouter>
      <ThemeContextProvider themeOptions={pmmThemeOptions}>
        {children}
      </ThemeContextProvider>
    </MemoryRouter>
  </AuthContext.Provider>
);

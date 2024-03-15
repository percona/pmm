import { ThemeContextProvider } from '@percona/design';
import { FC, PropsWithChildren } from 'react';
import { MemoryRouter } from 'react-router-dom';
import pmmThemeOptions from 'themes/PmmTheme';

export const TestWrapper: FC<PropsWithChildren> = ({ children }) => (
  <MemoryRouter>
    <ThemeContextProvider themeOptions={pmmThemeOptions}>
      {children}
    </ThemeContextProvider>
  </MemoryRouter>
);

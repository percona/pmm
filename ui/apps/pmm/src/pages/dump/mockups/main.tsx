// Standalone entry for the PMM Dump UX mockups (pmm-dump-mockups.html). It runs
// without a PMM Server: no API calls, a stubbed admin user and fixture data.
import { LocalizationProvider } from '@mui/x-date-pickers';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFnsV3';
import { pmmThemeOptions, ThemeContextProvider } from '@percona/peak-ui';
import { AuthContext } from 'contexts/auth';
import { UserContext } from 'contexts/user';
import React from 'react';
import ReactDOM from 'react-dom/client';
import { TEST_USER_ADMIN } from 'utils/testStubs';
import { PmmDumpMockups } from './PmmDumpMockups';

import '@fontsource/roboto/300.css';
import '@fontsource/roboto/400.css';
import '@fontsource/roboto/500.css';
import '@fontsource/roboto/700.css';
import '@fontsource/poppins/400.css';
import '@fontsource/poppins/500.css';
import '@fontsource/poppins/600.css';
import '@fontsource/roboto-mono';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <AuthContext.Provider value={{ isLoading: false, isLoggedIn: true }}>
      <UserContext.Provider value={{ isLoading: false, user: TEST_USER_ADMIN }}>
        <ThemeContextProvider themeOptions={pmmThemeOptions}>
          <LocalizationProvider dateAdapter={AdapterDateFns}>
            <PmmDumpMockups />
          </LocalizationProvider>
        </ThemeContextProvider>
      </UserContext.Provider>
    </AuthContext.Provider>
  </React.StrictMode>
);

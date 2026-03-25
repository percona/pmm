import { GlobalStyles } from '@mui/material';
import { AuthProvider } from 'contexts/auth';
import { GrafanaProvider } from 'contexts/grafana';
import { NavigationProvider } from 'contexts/navigation';
import { SettingsProvider } from 'contexts/settings';
import { Outlet } from 'react-router-dom';
import { UpdatesProvider } from 'contexts/updates';
import { UserProvider } from 'contexts/user';
import { FC, PropsWithChildren } from 'react';
import { useThemeSync } from 'hooks/useThemeSync';

const ThemeSyncProvider: FC<PropsWithChildren> = ({ children }) => {
  useThemeSync();
  return <>{children}</>;
};

const Providers: FC<PropsWithChildren> = () => (
  <AuthProvider>
    <UserProvider>
      <ThemeSyncProvider>
        <SettingsProvider>
          <UpdatesProvider>
            <GrafanaProvider>
              <NavigationProvider>
                  <GlobalStyles
                    styles={{
                      'html, body, div#root': {
                        minHeight: '100vh',
                      },
                      'div#root': {
                        display: 'flex',
                      },
                    }}
                  />
                  <Outlet />
              </NavigationProvider>
            </GrafanaProvider>
          </UpdatesProvider>
        </SettingsProvider>
      </ThemeSyncProvider>
    </UserProvider>
  </AuthProvider>
);

export default Providers;

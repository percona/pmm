import { GlobalStyles } from '@mui/material';
import { AuthProvider } from 'contexts/auth';
import { GrafanaProvider } from 'contexts/grafana';
import { NavigationProvider } from 'contexts/navigation';
import { SettingsProvider } from 'contexts/settings';
import { TourProvider } from 'contexts/tour';
import { UpdatesProvider } from 'contexts/updates';
import { UserProvider } from 'contexts/user';
import { VersionProvider } from 'contexts/version';
import { FC, PropsWithChildren } from 'react';
import { Outlet } from 'react-router-dom';
import { useThemeSync } from 'hooks/useThemeSync';

const ThemeSyncProvider: FC<PropsWithChildren> = ({ children }) => {
  useThemeSync();
  return <>{children}</>;
};

const Providers: FC<PropsWithChildren> = () => (
  <AuthProvider>
    <VersionProvider>
      <UserProvider>
        <ThemeSyncProvider>
          <SettingsProvider>
            <UpdatesProvider>
              <GrafanaProvider>
                <NavigationProvider>
                  <TourProvider>
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
                  </TourProvider>
                </NavigationProvider>
              </GrafanaProvider>
            </UpdatesProvider>
          </SettingsProvider>
        </ThemeSyncProvider>
      </UserProvider>
    </VersionProvider>
  </AuthProvider>
);

export default Providers;

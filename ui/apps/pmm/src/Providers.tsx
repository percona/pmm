import { Box, GlobalStyles } from '@mui/material';
import { AuthProvider } from 'contexts/auth';
import { GrafanaProvider } from 'contexts/grafana';
import { NavigationProvider } from 'contexts/navigation';
import { SettingsProvider } from 'contexts/settings';
import { TourProvider } from 'contexts/tour';
import { UpdatesProvider } from 'contexts/updates';
import { UserProvider } from 'contexts/user';
import { FC, PropsWithChildren } from 'react';
import { Outlet } from 'react-router-dom';
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
                <TourProvider>
                  <GlobalStyles
                    styles={{
                      'html, body, div#root': {
                        minHeight: '100vh',
                      },
                      'div#root': {
                        display: 'flex',
                        flex: 1,
                        flexDirection: 'column',
                        minHeight: 0,
                        width: '100%',
                      },
                    }}
                  />
                  <Box
                    component="div"
                    sx={{
                      flex: 1,
                      minHeight: 0,
                      minWidth: 0,
                      display: 'flex',
                      flexDirection: 'column',
                      width: '100%',
                    }}
                  >
                    <Outlet />
                  </Box>
                </TourProvider>
              </NavigationProvider>
            </GrafanaProvider>
          </UpdatesProvider>
        </SettingsProvider>
      </ThemeSyncProvider>
    </UserProvider>
  </AuthProvider>
);

export default Providers;

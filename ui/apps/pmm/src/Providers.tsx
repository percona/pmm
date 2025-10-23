import { GlobalStyles } from '@mui/material';
import { AuthProvider } from 'contexts/auth';
import { GrafanaProvider } from 'contexts/grafana';
import { NavigationProvider } from 'contexts/navigation';
import { SettingsProvider } from 'contexts/settings';
import { TourProvider } from 'contexts/tour';
import { UpdatesProvider } from 'contexts/updates';
import { UserProvider } from 'contexts/user';
import { FC, PropsWithChildren } from 'react';
import { Outlet } from 'react-router-dom';

const Providers: FC<PropsWithChildren> = () => (
  <AuthProvider>
    <UserProvider>
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
    </UserProvider>
  </AuthProvider>
);

export default Providers;

import { GlobalStyles } from '@mui/material';
import { AuthProvider } from 'contexts/auth';
import { GrafanaProvider } from 'contexts/grafana';
import { NavigationProvider } from 'contexts/navigation';
import { UpdatesProvider } from 'contexts/updates';
import { UserProvider } from 'contexts/user';
import { FC, PropsWithChildren } from 'react';
import { Outlet } from 'react-router-dom';

const Providers: FC<PropsWithChildren> = () => (
  <AuthProvider>
    <UserProvider>
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
    </UserProvider>
  </AuthProvider>
);

export default Providers;

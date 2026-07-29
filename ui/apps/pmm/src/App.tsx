import { LocalizationProvider } from '@mui/x-date-pickers';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFnsV3';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { RouterProvider } from 'react-router-dom';
import router from './router';
import { SnackbarProvider, CustomContentProps } from 'notistack';
import {
  ThemeContextProvider,
  pmmThemeOptions,
  NotistackMuiSnackbar,
} from '@percona/percona-ui';
import { ThemeClass } from 'components/theme-class';
import { useEffect } from 'react';
import type { ComponentType } from 'react';
import { addApiErrorInterceptor, removeApiErrorInterceptor } from 'api/api';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: false,
    },
  },
});

const App = () => {
  useEffect(() => {
    addApiErrorInterceptor();
    return () => {
      removeApiErrorInterceptor();
    };
  }, []);

  return (
    <ThemeContextProvider themeOptions={pmmThemeOptions}>
      <ThemeClass />
      <LocalizationProvider dateAdapter={AdapterDateFns}>
        <SnackbarProvider
          maxSnack={3}
          preventDuplicate
          // NOTE: using custom components disables notistack's custom actions, as per docs: https://notistack.com/features/basic#actions
          // If we need actions, we can add them to our custom component via useSnackbar(): https://notistack.com/features/customization#custom-component
          Components={{
            success: NotistackMuiSnackbar as ComponentType<CustomContentProps>,
            error: NotistackMuiSnackbar as ComponentType<CustomContentProps>,
            info: NotistackMuiSnackbar as ComponentType<CustomContentProps>,
            warning: NotistackMuiSnackbar as ComponentType<CustomContentProps>,
          }}
          // Render the snackbar on the right side of the screen to not interfere with navigation
          anchorOrigin={{ horizontal: 'right', vertical: 'bottom' }}
        >
          <QueryClientProvider client={queryClient}>
            <RouterProvider router={router} />
          </QueryClientProvider>
        </SnackbarProvider>
      </LocalizationProvider>
    </ThemeContextProvider>
  );
};

export default App;

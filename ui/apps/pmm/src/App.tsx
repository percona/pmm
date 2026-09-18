import type { PaletteMode } from '@mui/material';
import type { ThemeOptions } from '@mui/material/styles';
import { LocalizationProvider } from '@mui/x-date-pickers';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFnsV3';
import {
  NotistackMuiSnackbar,
  ThemeContextProvider,
  pmmThemeOptions,
} from '@percona/peak-ui';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { addApiErrorInterceptor, removeApiErrorInterceptor } from 'api/api';
import { ThemeClass } from 'components/theme-class';
import { CustomContentProps, SnackbarProvider } from 'notistack';
import type { ComponentType } from 'react';
import { useEffect } from 'react';
import { RouterProvider } from 'react-router-dom';
import router from './router';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: false,
    },
  },
});

const DARK_BACKGROUND_COLOR = 'rgb(10, 10, 18)';

// pmmThemeOptions with the dark background overridden; `paper` is included
// because most page surfaces (RTA, Grafana frame, settings) paint with it
const themeOptions = (mode: PaletteMode): ThemeOptions => {
  const options = pmmThemeOptions(mode);

  if (mode !== 'dark') {
    return options;
  }

  return {
    ...options,
    palette: {
      ...options.palette,
      background: {
        ...options.palette?.background,
        default: DARK_BACKGROUND_COLOR,
        paper: DARK_BACKGROUND_COLOR,
      },
    },
  };
};

const App = () => {
  useEffect(() => {
    addApiErrorInterceptor();
    return () => {
      removeApiErrorInterceptor();
    };
  }, []);

  return (
    <ThemeContextProvider themeOptions={themeOptions}>
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

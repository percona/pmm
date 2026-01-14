import { LocalizationProvider } from '@mui/x-date-pickers';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { RouterProvider } from 'react-router-dom';
import router from './router';
import { SnackbarProvider, CustomContentProps } from 'notistack';
import {
  ThemeContextProvider,
  pmmThemeOptions,
  NotistackMuiSnackbar,
} from '@percona/percona-ui';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
    },
  },
});

const App = () => (
  <ThemeContextProvider
    themeOptions={pmmThemeOptions}
    saveColorModeOnLocalStorage
  >
    <LocalizationProvider dateAdapter={AdapterDateFns}>
      <SnackbarProvider
        maxSnack={3}
        preventDuplicate
        // NOTE: using custom components disables notistack's custom actions, as per docs: https://notistack.com/features/basic#actions
        // If we need actions, we can add them to our custom component via useSnackbar(): https://notistack.com/features/customization#custom-component
        Components={{
          success:
            NotistackMuiSnackbar as React.ComponentType<CustomContentProps>,
          error:
            NotistackMuiSnackbar as React.ComponentType<CustomContentProps>,
          info: NotistackMuiSnackbar as React.ComponentType<CustomContentProps>,
          warning:
            NotistackMuiSnackbar as React.ComponentType<CustomContentProps>,
        }}
      >
        <QueryClientProvider client={queryClient}>
          <RouterProvider router={router} />
        </QueryClientProvider>
      </SnackbarProvider>
    </LocalizationProvider>
  </ThemeContextProvider>
);

export default App;

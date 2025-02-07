import { LocalizationProvider } from '@mui/x-date-pickers';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { RouterProvider } from 'react-router-dom';
import router from './router';
import { ThemeContextProvider } from '@percona/design';
import { NotistackMuiSnackbar } from '@percona/ui-lib';
import { SnackbarProvider } from 'notistack';
import pmmThemeOptions from 'themes/PmmTheme';
import { AuthProvider } from 'contexts/auth';
import { UserProvider } from 'contexts/user';
import { UpdatesProvider } from 'contexts/updates';
import { NavigationProvider } from 'contexts/navigation/navigation.provider';

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
          success: NotistackMuiSnackbar,
          error: NotistackMuiSnackbar,
          info: NotistackMuiSnackbar,
          warning: NotistackMuiSnackbar,
        }}
      >
        <QueryClientProvider client={queryClient}>
          <AuthProvider>
            <UserProvider>
              <UpdatesProvider>
                <NavigationProvider>
                  <RouterProvider router={router} />
                </NavigationProvider>
              </UpdatesProvider>
            </UserProvider>
          </AuthProvider>
        </QueryClientProvider>
      </SnackbarProvider>
    </LocalizationProvider>
  </ThemeContextProvider>
);

export default App;

import { AxiosError, AxiosInstance, HttpStatusCode } from 'axios';
import { enqueueSnackbar } from 'notistack';
import { isClientSessionEstablished } from 'contexts/auth/auth.clientSession';
import { redirectToLogin } from 'contexts/auth/auth.utils';
import { api, grafanaApi } from './api';
import { getApiErrorMessage } from './api.utils';
import { ROTATE_TOKEN_URL, rotateToken } from './auth';

const recoverFromUnauthorized =
  (instance: AxiosInstance) => async (error: AxiosError) => {
    const config = error.config;

    if (
      error.response?.status !== HttpStatusCode.Unauthorized ||
      !config ||
      config.authRetried ||
      // rotating in response to a failed rotation would wait on its own request
      config.url === ROTATE_TOKEN_URL
    ) {
      return Promise.reject(error);
    }

    config.authRetried = true;

    try {
      await rotateToken();
    } catch (rotationError) {
      // The rotate endpoint only refuses once the session is past
      // `login_maximum_inactive_lifetime_duration`, at which point it cannot be recovered.
      // Anonymous access never held a session to lose, so leave it where it is.
      if (
        (rotationError as AxiosError).response?.status ===
          HttpStatusCode.Unauthorized &&
        isClientSessionEstablished()
      ) {
        redirectToLogin();
      }

      return Promise.reject(error);
    }

    // A replay that fails again is not an expired session — pmm-managed also answers 401
    // for internal errors — so let it fall through to the error notifier.
    return instance.request(config);
  };

const notifyError = (error: AxiosError<{ message?: string }>) => {
  if (error.response && error.response.status >= 400) {
    let notificationsDisabled =
      error.config?.disableNotifications ?? error.response.status === 429;

    if (typeof notificationsDisabled === 'function') {
      notificationsDisabled = notificationsDisabled(error);
    }

    if (!notificationsDisabled) {
      enqueueSnackbar(getApiErrorMessage(error), {
        variant: 'error',
        preventDuplicate: true,
      });
      error.notified = true;
    }
  }

  return Promise.reject(error);
};

// Registered at module scope so every request is covered
for (const instance of [api, grafanaApi]) {
  instance.interceptors.response.use(
    (response) => response,
    recoverFromUnauthorized(instance)
  );
}

api.interceptors.response.use((response) => response, notifyError);

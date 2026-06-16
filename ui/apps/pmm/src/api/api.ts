import axios, { AxiosError } from 'axios';
import applyCaseMiddleware from 'axios-case-converter';
import { enqueueSnackbar } from 'notistack';


export const api = applyCaseMiddleware(
  axios.create({
    baseURL: '/v1/',
  })
);

export const grafanaApi = axios.create({
  baseURL: '/graph/api/',
});

const DEFAULT_ERROR_MESSAGE = 'Something went wrong';
const MAX_ERROR_MESSAGE_LENGTH = 120;
let apiErrorInterceptor: number | null = null;

const onApiError = (error: AxiosError<{ message?: string }>) => {
  if (
    error.response &&
    error.response.status >= 400
  ) {
    let message = error.response.data?.message ?? DEFAULT_ERROR_MESSAGE;
    let notificationsDisabled =
      error.config?.disableNotifications ?? error.response.status === 429;

    if (typeof notificationsDisabled === 'function') {
      notificationsDisabled = notificationsDisabled(error);
    }

    if (!notificationsDisabled) {
      message = message.trim();
      if (message.length > MAX_ERROR_MESSAGE_LENGTH) {
        message = `${message.substring(0, MAX_ERROR_MESSAGE_LENGTH)}...`;
      }

      enqueueSnackbar(message, {
        variant: 'error',
        preventDuplicate: true,
      });
    }
  }

  return Promise.reject(error);
};

export const addApiErrorInterceptor = () => {
  if (apiErrorInterceptor === null) {
    apiErrorInterceptor = api.interceptors.response.use(
      (response) => response,
      onApiError
    );
  }

};

export const removeApiErrorInterceptor = () => {
  if (apiErrorInterceptor !== null) {
    api.interceptors.response.eject(apiErrorInterceptor);
    apiErrorInterceptor = null;
  }

};

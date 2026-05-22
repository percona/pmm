import type { AxiosError } from 'axios';
import type { DefaultError, QueryKey } from '@tanstack/react-query';

export interface ApiErrorResponse {
  error: string;
  code: number;
  message: string;
}

declare module 'axios' {
  export interface AxiosRequestConfig {
    disableNotifications?: boolean | ((error: AxiosError) => boolean);
  }
}

declare module '@tanstack/react-query' {
  /* eslint-disable @typescript-eslint/no-unused-vars */
  interface UseQueryOptions<
    TQueryFnData = unknown,
    TError = DefaultError,
    TData = TQueryFnData,
    TQueryKey extends QueryKey = QueryKey,
  > {
    axios?: import('axios').AxiosRequestConfig;
  }
  /* eslint-enable @typescript-eslint/no-unused-vars */
}

export interface ApiError extends AxiosError<ApiErrorResponse> {}

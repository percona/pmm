import type { AxiosError } from 'axios';

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
  interface UseQueryOptions {
    axios?: import('axios').AxiosRequestConfig;
  }
}

export interface ApiError extends AxiosError<ApiErrorResponse> {}

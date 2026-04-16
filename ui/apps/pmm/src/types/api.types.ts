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

export interface ApiError extends AxiosError<ApiErrorResponse> {}

import { AxiosError } from 'axios';

export interface ApiErrorResponse {
  error: string;
  code: number;
  message: string;
}

export interface ApiError extends AxiosError<ApiErrorResponse> {}

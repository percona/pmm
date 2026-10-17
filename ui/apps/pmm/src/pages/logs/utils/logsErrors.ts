import { isAxiosError } from 'axios';

export const LBAC_MESSAGE =
  'Logs are not available to users with label-based access filters.';

// gRPC status code returned in the error body.
const PERMISSION_DENIED = 7;

export const logsErrorMessage = (error: unknown): string => {
  if (isAxiosError<{ code?: number; message?: string }>(error)) {
    const data = error.response?.data;
    if (data?.code === PERMISSION_DENIED) return LBAC_MESSAGE;
    if (data?.message) return data.message;
  }
  return 'Failed to load logs.';
};

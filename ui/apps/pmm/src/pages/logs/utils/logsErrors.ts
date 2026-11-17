import { isAxiosError } from 'axios';

export const LBAC_MESSAGE =
  'Logs are not available to users with label-based access filters.';

// The service explains denials itself; nginx's role denial carries the same code with "Access denied".
export const logsErrorMessage = (error: unknown): string => {
  if (isAxiosError<{ message?: string; error?: string }>(error)) {
    const data = error.response?.data;
    if (data?.message) return data.message;
    if (data?.error) return data.error;
  }
  return 'Failed to load logs.';
};

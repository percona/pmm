import type { ApiError } from 'types/api.types';

export function apiErrorMessage(err: unknown, fallback = 'Something went wrong'): string {
  const ax = err as ApiError;
  const msg = ax?.response?.data?.message?.trim() || ax?.response?.data?.error?.trim();
  if (msg) return msg;
  if (err instanceof Error && err.message) return err.message;
  return fallback;
}

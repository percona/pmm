import { useMutation, UseMutationOptions } from '@tanstack/react-query';
import { createShortUrl } from 'api/short-urls';
import { ApiError } from 'types/api.types';
import { CreateShortUrlResponse } from 'types/short-urls.types';

export const useCreateShortUrl = (
  options?: UseMutationOptions<CreateShortUrlResponse, ApiError, string>
) =>
  useMutation({
    mutationKey: ['short-urls:create'],
    mutationFn: (path: string) => createShortUrl(path),
    ...options,
  });

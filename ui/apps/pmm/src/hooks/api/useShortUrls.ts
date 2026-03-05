import { useMutation, UseMutationOptions } from '@tanstack/react-query';
import { createShortUrl } from 'api/short-urls';
import { ApiError } from 'types/api.types';
import { CreateShortUrlResponse } from 'types/short-urls.types';

const buildHostUrl = () => {
  const iframe = document.getElementById(
    'grafana-iframe'
  ) as HTMLIFrameElement | null;
  if (!iframe) {
    throw new Error('Grafana iframe not found');
  }
  // @ts-ignore
  const config = iframe.contentWindow?.grafanaBootData;
  return `${window.location.protocol}//${window.location.host}${config.settings.appSubUrl}`;
};

const getRelativeURLPath = (url: string) => {
  const path = url.replace(buildHostUrl(), '');
  return path.startsWith('/') ? path.substring(1, path.length) : path;
};

export const useCreateShortUrl = (
  options?: UseMutationOptions<CreateShortUrlResponse, ApiError, string>
) =>
  useMutation({
    mutationKey: ['short-urls:create'],
    mutationFn: (path: string) => createShortUrl(getRelativeURLPath(path)),
    ...options,
  });

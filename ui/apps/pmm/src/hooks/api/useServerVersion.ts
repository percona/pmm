import { useQuery, UseQueryOptions } from '@tanstack/react-query';
import { getServerVersion } from 'api/server';
import { INTERVALS_MS } from 'lib/constants';
import { GetServerVersionResponse } from 'types/server.types';

export const SERVER_VERSION_QUERY_KEY = ['server:version'] as const;

export const useServerVersion = (
  options?: Partial<UseQueryOptions<GetServerVersionResponse>>
) =>
  useQuery({
    queryKey: SERVER_VERSION_QUERY_KEY,
    queryFn: getServerVersion,
    refetchInterval: INTERVALS_MS.SERVER_VERSION,
    refetchIntervalInBackground: true,
    retry: false,
    ...options,
  });

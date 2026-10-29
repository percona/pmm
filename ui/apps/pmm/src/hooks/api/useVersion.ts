import { useQuery, UseQueryOptions } from '@tanstack/react-query';
import { getVersion } from 'api/version';
import { VersionResponse } from 'types/version.types';

export const useVersion = (
  options?: Partial<UseQueryOptions<VersionResponse>>
) =>
  useQuery({
    queryKey: ['server:version'],
    queryFn: () => getVersion(),
    ...options,
  });

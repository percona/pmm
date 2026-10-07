/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

import { QueryClient, type QueryClientConfig } from '@tanstack/react-query';
import { AxiosError } from 'axios';
import { ApiError } from './errors';

const DEFAULT_STALE_TIME_MS = 30_000;
const MAX_RETRIES = 3;
const RETRY_BASE_DELAY_MS = 500;
const RETRY_MAX_DELAY_MS = 30_000;

function isClientError(error: unknown): boolean {
  if (error instanceof ApiError) {
    return (
      error.kind === 'http' &&
      (error.status ?? 0) >= 400 &&
      (error.status ?? 0) < 500
    );
  }
  if (error instanceof AxiosError) {
    const status = error.response?.status;
    return typeof status === 'number' && status >= 400 && status < 500;
  }
  return false;
}

function shouldRetry(failureCount: number, error: unknown): boolean {
  if (failureCount >= MAX_RETRIES) {
    return false;
  }
  if (isClientError(error)) {
    return false;
  }
  return true;
}

function retryDelay(attemptIndex: number): number {
  return Math.min(RETRY_BASE_DELAY_MS * 2 ** attemptIndex, RETRY_MAX_DELAY_MS);
}

export const defaultQueryClientConfig: QueryClientConfig = {
  defaultOptions: {
    queries: {
      staleTime: DEFAULT_STALE_TIME_MS,
      refetchOnWindowFocus: true,
      retry: shouldRetry,
      retryDelay,
    },
    mutations: {
      retry: false,
    },
  },
};

export function createQueryClient(overrides?: QueryClientConfig): QueryClient {
  if (!overrides) {
    return new QueryClient(defaultQueryClientConfig);
  }
  return new QueryClient({
    ...defaultQueryClientConfig,
    ...overrides,
    defaultOptions: {
      ...defaultQueryClientConfig.defaultOptions,
      ...overrides.defaultOptions,
      queries: {
        ...defaultQueryClientConfig.defaultOptions?.queries,
        ...overrides.defaultOptions?.queries,
      },
      mutations: {
        ...defaultQueryClientConfig.defaultOptions?.mutations,
        ...overrides.defaultOptions?.mutations,
      },
    },
  });
}

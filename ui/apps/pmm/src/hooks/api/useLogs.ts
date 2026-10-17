import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import {
  getLogContext,
  getLogFacets,
  getLogPatterns,
  getLogVolume,
  LogFilter,
  LogLine,
  searchLogs,
} from 'api/logs';

export const LOGS_KEYS = {
  all: ['logs'] as const,
  search: (f: LogFilter) => ['logs', 'search', f] as const,
  volume: (f: LogFilter) => ['logs', 'volume', f] as const,
  patterns: (f: LogFilter) => ['logs', 'patterns', f] as const,
  facets: (start: string, end: string) =>
    ['logs', 'facets', start, end] as const,
  context: (line: LogLine | null) =>
    ['logs', 'context', line?.time, line?.nodeName, line?.fileName] as const,
};

/** Pages through matching lines, newest first, following nextCursor. */
export const useLogSearch = (filter: LogFilter) =>
  useInfiniteQuery({
    queryKey: LOGS_KEYS.search(filter),
    queryFn: ({ pageParam }) => searchLogs(filter, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.nextCursor,
  });

export const useLogVolume = (filter: LogFilter) =>
  useQuery({
    queryKey: LOGS_KEYS.volume(filter),
    queryFn: () => getLogVolume(filter),
  });

export const useLogPatterns = (filter: LogFilter, enabled: boolean) =>
  useQuery({
    queryKey: LOGS_KEYS.patterns(filter),
    queryFn: () => getLogPatterns(filter),
    enabled,
  });

/** Filter values over the time range only, so the pickers list everything in it. */
export const useLogFacets = (filter: LogFilter) =>
  useQuery({
    queryKey: LOGS_KEYS.facets(filter.start, filter.end),
    queryFn: () => getLogFacets({ start: filter.start, end: filter.end }),
  });

export const useLogContext = (line: LogLine | null) =>
  useQuery({
    queryKey: LOGS_KEYS.context(line),
    queryFn: () => getLogContext(line!.time, line!.nodeName, line!.fileName),
    enabled: !!line,
  });

import { RealTimeQueryData, QueryState } from 'types/realtime.types';

export const formatDuration = (duration: number): string => {
  if (duration < 1000) {
    return `${duration.toFixed(0)}ms`;
  }
  if (duration < 60000) {
    return `${(duration / 1000).toFixed(1)}s`;
  }
  return `${(duration / 60000).toFixed(1)}m`;
};

export const formatTimestamp = (timestamp: string): string => {
  return new Date(timestamp).toLocaleString();
};

export const getQueryStateColor = (state: QueryState): string => {
  switch (state) {
    case QueryState.RUNNING:
      return '#4caf50'; // Green
    case QueryState.WAITING:
      return '#ff9800'; // Orange
    case QueryState.FINISHED:
      return '#2196f3'; // Blue
    case QueryState.UNKNOWN:
    default:
      return '#9e9e9e'; // Gray
  }
};

export const getQueryStateLabel = (state: QueryState): string => {
  switch (state) {
    case QueryState.RUNNING:
      return 'Running';
    case QueryState.WAITING:
      return 'Waiting';
    case QueryState.FINISHED:
      return 'Finished';
    case QueryState.UNKNOWN:
    default:
      return 'Unknown';
  }
};

export const truncateQueryText = (queryText: string, maxLength = 100): string => {
  if (queryText.length <= maxLength) {
    return queryText;
  }
  return `${queryText.substring(0, maxLength)}...`;
};

export const formatQueryText = (queryText?: string): string => {
  if (!queryText) {
    return '';
  }
  try {
    // Try to format JSON queries
    const parsed = JSON.parse(queryText);
    return JSON.stringify(parsed, null, 2);
  } catch {
    // If not JSON, return as is
    return queryText;
  }
};

export const getQueryComplexity = (queryData: RealTimeQueryData): 'simple' | 'medium' | 'complex' => {
  const queryText = queryData.queryText || '';
  const queryLength = queryText.length;
  const hasAggregation = queryText.includes('$group') || queryText.includes('$match');
  const hasJoins = queryText.includes('$lookup');
  
  if (queryLength > 500 || hasJoins) {
    return 'complex';
  }
  if (queryLength > 200 || hasAggregation) {
    return 'medium';
  }
  return 'simple';
};

export const sortQueriesByDuration = (queries: RealTimeQueryData[]): RealTimeQueryData[] => {
  return [...queries].sort((a, b) => {
    const aDuration = a.currentExecutionTime || 0;
    const bDuration = b.currentExecutionTime || 0;
    return bDuration - aDuration;
  });
};

export const filterQueriesByState = (queries: RealTimeQueryData[], state: QueryState): RealTimeQueryData[] => {
  return queries.filter(query => query.state === state);
};

export const searchQueries = (queries: RealTimeQueryData[], searchTerm: string): RealTimeQueryData[] => {
  if (!searchTerm.trim()) {
    return queries;
  }
  
  const term = searchTerm.toLowerCase();
  return queries.filter(query => 
    (query.database || '').toLowerCase().includes(term) ||
    (query.mongodb?.operationType || '').toLowerCase().includes(term) ||
    (query.fingerprint || '').toLowerCase().includes(term) ||
    (query.queryText || '').toLowerCase().includes(term)
  );
};

export const deduplicateQueriesByOpId = (queries: RealTimeQueryData[]): RealTimeQueryData[] => {
  const queryIdToQuery = new Map<string, RealTimeQueryData>();
  
  // Group queries by queryId (which is now the opid) and keep the one with the latest timestamp
  // This handles cases where the same operation appears multiple times with different collection timestamps
  for (const query of queries) {
    const queryId = query.queryId;
    
    if (queryId) {
      const existingQuery = queryIdToQuery.get(queryId);
      if (!existingQuery || new Date(query.timestamp) > new Date(existingQuery.timestamp)) {
        // Keep this query if it's the first with this queryId or has a later timestamp
        queryIdToQuery.set(queryId, query);
      }
    }
    // Note: All queries should now have a queryId since it's the opid
  }
  
  return Array.from(queryIdToQuery.values());
};

export const getQueryOpId = (query: RealTimeQueryData): string | undefined => {
  // queryId is now the opid, so we can return it directly
  return query.queryId || query.mongodb?.opid?.toString(); // Convert number to string
};

export const groupQueriesByOpId = (queries: RealTimeQueryData[]): Map<string, RealTimeQueryData[]> => {
  const groups = new Map<string, RealTimeQueryData[]>();
  
  for (const query of queries) {
    const opid = getQueryOpId(query);
    if (opid) {
      if (!groups.has(opid)) {
        groups.set(opid, []);
      }
      groups.get(opid)!.push(query);
    }
  }
  
  return groups;
};

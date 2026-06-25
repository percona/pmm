import { type MRT_Row } from 'material-react-table';
import { isPostgresQuery, QueryData } from 'types/rta.types';

export const filterElapsedTime = (
  row: MRT_Row<QueryData>,
  id: string,
  filterValue: [string, string]
) => {
  const [min, max] = filterValue;
  const valueSeconds = row.getValue<number>(id);
  if (valueSeconds === null || valueSeconds === undefined) return false;

  const minSet = min !== '' && min != null && !Number.isNaN(parseFloat(min));
  const maxSet = max !== '' && max != null && !Number.isNaN(parseFloat(max));

  if (!minSet && !maxSet) return true;

  if (minSet && !maxSet) return valueSeconds >= parseFloat(min);
  if (!minSet && maxSet) return valueSeconds <= parseFloat(max);

  return valueSeconds >= parseFloat(min) && valueSeconds <= parseFloat(max);
};

export const hasPostgresQueries = (queries: QueryData[]): boolean =>
  queries.some(isPostgresQuery);

/** Hide parallel workers; attach worker count to leader rows for PG 13+. */
export const prepareOverviewQueries = (queries: QueryData[]): QueryData[] => {
  const workersByLeader = new Map<number, number>();

  for (const query of queries) {
    const leaderPid = query.postgresPayload?.leaderPid ?? 0;
    if (leaderPid > 0) {
      workersByLeader.set(leaderPid, (workersByLeader.get(leaderPid) ?? 0) + 1);
    }
  }

  return queries
    .filter((query) => (query.postgresPayload?.leaderPid ?? 0) === 0)
    .map((query) => {
      const backendPid = query.postgresPayload?.backendPid ?? 0;
      const workerCount = backendPid > 0 ? workersByLeader.get(backendPid) : undefined;

      if (!workerCount) {
        return query;
      }

      return {
        ...query,
        postgresPayload: {
          ...query.postgresPayload!,
          parallelWorkerCount: workerCount,
        },
      };
    });
};

export const PG_READ_ALL_STATS_ERROR =
  'Monitoring user lacks pg_read_all_stats role. Grant pg_read_all_stats or use a superuser account for full pg_stat_activity visibility.';

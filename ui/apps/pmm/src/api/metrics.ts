import type {
  DataFrameJSON,
  DataQueryResponse,
  InstantQuery,
  InstantSample,
} from 'types/metrics.types';
import { grafanaApi } from './api';

// Over this range and resolution, $__rate_interval resolves to about one minute.
const QUERY_RANGE = { from: 'now-1h', to: 'now' };
const MAX_DATA_POINTS = 60;
const INTERVAL_MS = 60_000;

const toNumber = (value: unknown) =>
  typeof value === 'number' && Number.isFinite(value) ? value : undefined;

const toSamples = (frames: DataFrameJSON[] = []): InstantSample[] =>
  frames.flatMap((frame) => {
    const fields = frame.schema?.fields ?? [];
    const index = fields.findIndex((field) => field.type === 'number');

    if (index < 0) {
      return [];
    }

    const values = frame.data?.values[index] ?? [];

    return [
      {
        labels: fields[index].labels ?? {},
        value: toNumber(values[values.length - 1]),
      },
    ];
  });

// Instant PromQL queries through Grafana, which filters them by LBAC and lets viewers run them.
export const queryInstant = async (
  datasourceUid: string,
  queries: InstantQuery[]
): Promise<Record<string, InstantSample[]>> => {
  const datasource = { type: 'prometheus', uid: datasourceUid };
  const res = await grafanaApi.post<DataQueryResponse>(
    '/ds/query',
    {
      ...QUERY_RANGE,
      queries: queries.map((query) => ({
        ...query,
        datasource,
        instant: true,
        range: false,
        format: 'table',
        maxDataPoints: MAX_DATA_POINTS,
        intervalMs: INTERVAL_MS,
      })),
    },
    // The pages show their own error state.
    { disableNotifications: true }
  );

  return Object.fromEntries(
    queries.map(({ refId }) => {
      const result = res.data.results[refId];

      if (result?.error) {
        throw new Error(result.error);
      }

      return [refId, toSamples(result?.frames)];
    })
  );
};

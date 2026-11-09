import type {
  DataFrameJSON,
  DataQueryResponse,
  InstantQuery,
  InstantSample,
} from 'types/metrics.types';
import { isAxiosError } from 'axios';
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

// Grafana answers 400 for the whole request when one query fails, with every result in the body.
const postQueries = async (body: object): Promise<DataQueryResponse> => {
  try {
    const res = await grafanaApi.post<DataQueryResponse>('/ds/query', body);

    return res.data;
  } catch (error) {
    if (
      isAxiosError<DataQueryResponse>(error) &&
      error.response?.status === 400 &&
      error.response.data?.results
    ) {
      return error.response.data;
    }

    throw error;
  }
};

// Instant PromQL queries through Grafana, which filters them by LBAC and lets viewers run them.
// A failed query fails the call when it is required, and returns no samples otherwise.
export const queryInstant = async (
  datasourceUid: string,
  queries: InstantQuery[],
  required: string[] = queries.map(({ refId }) => refId)
): Promise<Record<string, InstantSample[]>> => {
  const datasource = { type: 'prometheus', uid: datasourceUid };
  const data = await postQueries({
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
  });

  return Object.fromEntries(
    queries.map(({ refId }) => {
      const result = data.results[refId];

      if (result?.error && required.includes(refId)) {
        throw new Error(result.error);
      }

      return [refId, result?.error ? [] : toSamples(result?.frames)];
    })
  );
};

// Through nginx's /prometheus/api/v1 location, so vmproxy applies label-based access control.
export const INSTANT_QUERY_URL = '/prometheus/api/v1/query';

export interface PromSample {
  metric: Record<string, string>;
  value: number;
  /** Epoch seconds of the sample. */
  timestamp: number;
}

interface PromVectorResponse {
  status: 'success' | 'error';
  errorType?: string;
  error?: string;
  data?: {
    resultType: string;
    result: Array<{ metric: Record<string, string>; value: [number, string] }>;
  };
}

export class PromQueryError extends Error {
  readonly status?: number;

  constructor(message: string, status?: number) {
    super(message);
    this.name = 'PromQueryError';
    this.status = status;
  }
}

/** Reads a Prometheus API body as an instant vector; any other shape throws. */
export const parseVector = (body: PromVectorResponse): PromSample[] => {
  if (body.status !== 'success') {
    throw new PromQueryError(body.error || body.errorType || 'Query failed');
  }
  if (body.data?.resultType !== 'vector') {
    throw new PromQueryError(
      `Expected a vector, got ${body.data?.resultType ?? 'nothing'}`
    );
  }

  return body.data.result.flatMap(({ metric, value: [timestamp, raw] }) => {
    const value = Number(raw);

    // NaN and ±Inf come from divisions by zero; they carry no reading.
    return Number.isFinite(value) ? [{ metric, value, timestamp }] : [];
  });
};

/** One instant query, `GET /prometheus/api/v1/query`; a failed query throws so it never reads as an empty result. */
export const instantQuery = async (
  query: string,
  init: { signal?: AbortSignal; time?: number } = {}
): Promise<PromSample[]> => {
  const params = new URLSearchParams({ query });

  if (init.time !== undefined) {
    params.set('time', String(init.time));
  }

  const response = await fetch(`${INSTANT_QUERY_URL}?${params.toString()}`, {
    credentials: 'same-origin',
    headers: { Accept: 'application/json' },
    signal: init.signal,
  });
  const body = (await response
    .json()
    .catch(() => null)) as PromVectorResponse | null;

  if (!response.ok || !body) {
    throw new PromQueryError(
      body?.error || `Request failed with ${response.status}`,
      response.status
    );
  }

  return parseVector(body);
};

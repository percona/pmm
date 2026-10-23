import {
  INSTANT_QUERY_URL,
  instantQuery,
  parseVector,
  PromQueryError,
} from './victoriaMetrics';

const jsonResponse = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });

// A vector as VictoriaMetrics answers an instant query.
const SHARDS_BODY = {
  status: 'success',
  data: {
    resultType: 'vector',
    result: [{ metric: { cluster: 'orders' }, value: [1792424620, '2'] }],
  },
  stats: { seriesFetched: '1', executionTimeMsec: 5 },
};

describe('parseVector', () => {
  it('reads metric labels, value and timestamp', () => {
    expect(parseVector(SHARDS_BODY as never)).toEqual([
      { metric: { cluster: 'orders' }, value: 2, timestamp: 1792424620 },
    ]);
  });

  it('returns an empty list for an empty vector', () => {
    expect(
      parseVector({
        status: 'success',
        data: { resultType: 'vector', result: [] },
      })
    ).toEqual([]);
  });

  it('drops NaN and infinite samples', () => {
    expect(
      parseVector({
        status: 'success',
        data: {
          resultType: 'vector',
          result: [
            { metric: { service_id: 'a' }, value: [1, 'NaN'] },
            { metric: { service_id: 'b' }, value: [1, '+Inf'] },
            { metric: { service_id: 'c' }, value: [1, '0.5'] },
          ],
        },
      })
    ).toEqual([{ metric: { service_id: 'c' }, value: 0.5, timestamp: 1 }]);
  });

  it('throws on status error, so a failed query never reads as no data', () => {
    expect(() =>
      parseVector({ status: 'error', errorType: '400', error: 'bad query' })
    ).toThrow(new PromQueryError('bad query'));
  });

  it('throws on a result that is not a vector', () => {
    expect(() =>
      parseVector({
        status: 'success',
        data: { resultType: 'scalar', result: [] },
      })
    ).toThrow(PromQueryError);
  });
});

describe('instantQuery', () => {
  const fetchMock = vi.fn();

  beforeEach(() => {
    vi.stubGlobal('fetch', fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    fetchMock.mockReset();
  });

  it('sends a same-origin GET with the query in the URL', async () => {
    fetchMock.mockResolvedValue(jsonResponse(SHARDS_BODY));

    const samples = await instantQuery('up{job=~".*_hr"}', { time: 1700 });
    const [url, init] = fetchMock.mock.calls[0];
    const parsed = new URL(url, 'https://pmm.test');

    expect(parsed.pathname).toBe(INSTANT_QUERY_URL);
    expect(parsed.searchParams.get('query')).toBe('up{job=~".*_hr"}');
    expect(parsed.searchParams.get('time')).toBe('1700');
    expect(init).toMatchObject({ credentials: 'same-origin' });
    expect(init.method).toBeUndefined();
    expect(samples).toHaveLength(1);
  });

  it('throws the server message on an HTTP error', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse(
        { status: 'error', errorType: '400', error: 'unexpected token' },
        400
      )
    );

    await expect(instantQuery('max by (')).rejects.toMatchObject({
      name: 'PromQueryError',
      message: 'unexpected token',
      status: 400,
    });
  });

  it('throws on a response that is not JSON', async () => {
    fetchMock.mockResolvedValue(
      new Response('<html>504</html>', { status: 504 })
    );

    await expect(instantQuery('up')).rejects.toMatchObject({
      message: 'Request failed with 504',
      status: 504,
    });
  });
});

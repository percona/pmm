import { getExternalUrlError, parseExternalUrl } from './ExternalForm.utils';
import { Scheme } from './ExternalForm.types';

describe('ExternalForm.utils', () => {
  it.each([
    ['http://redis-exporter:9121/metrics', undefined],
    ['https://example.com', undefined],
    ['http://[fd00::5]:9121/metrics', undefined],
    ['ftp://example.com', 'Invalid URL string'],
    ['example.com:9121', 'Invalid URL string'],
    ['', 'Invalid URL string'],
    [
      'http://exporter:9121/metrics?target=redis',
      'Query parameters are not supported in the endpoint',
    ],
  ])('validates %j', (url, error) => {
    expect(getExternalUrlError(url)).toBe(error);
  });

  it('sends an IPv6 host without its URL brackets', () => {
    expect(parseExternalUrl('http://[fd00::5]:9121/metrics').address).toBe(
      'fd00::5'
    );
  });

  it('reads the endpoint from the URL', () => {
    expect(
      parseExternalUrl('http://user:p%40ss@redis-exporter:9121/metrics')
    ).toEqual({
      scheme: Scheme.http,
      address: 'redis-exporter',
      listenPort: 9121,
      metricsPath: '/metrics',
      username: 'user',
      password: 'p@ss',
    });
  });

  it('falls back to the scheme port and the default metrics path', () => {
    expect(parseExternalUrl('https://example.com')).toEqual(
      expect.objectContaining({
        listenPort: 443,
        metricsPath: undefined,
      })
    );
    expect(parseExternalUrl('http://example.com').listenPort).toBe(80);
  });
});

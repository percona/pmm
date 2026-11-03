import { isValidExternalUrl, parseExternalUrl } from './ExternalForm.utils';
import { Scheme } from './ExternalForm.types';

describe('ExternalForm.utils', () => {
  it.each([
    ['http://redis-exporter:9121/metrics', true],
    ['https://example.com', true],
    ['ftp://example.com', false],
    ['example.com:9121', false],
    ['', false],
  ])('validates %j', (url, valid) => {
    expect(isValidExternalUrl(url)).toBe(valid);
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

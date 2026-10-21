import { get } from './api';
import { checkForUpdates } from './updates';

jest.mock('./api');

describe('checkForUpdates', () => {
  it('should map the response to camelCase', async () => {
    jest.mocked(get).mockResolvedValue({
      installed: {
        version: '3.4.0',
        full_version: '3.4.0-1',
        timestamp: '2025-01-01T00:00:00Z',
      },
      latest: {
        version: '3.5.0',
        tag: 'tag',
        timestamp: '2025-02-01T00:00:00Z',
      },
      update_available: true,
      latest_news_url: 'https://percona.com',
      last_check: '2025-02-02T00:00:00Z',
    });

    expect(await checkForUpdates()).toEqual({
      installed: {
        version: '3.4.0',
        fullVersion: '3.4.0-1',
        timestamp: '2025-01-01T00:00:00Z',
      },
      latest: {
        version: '3.5.0',
        tag: 'tag',
        timestamp: '2025-02-01T00:00:00Z',
      },
      updateAvailable: true,
      latestNewsUrl: 'https://percona.com',
      lastChecked: '2025-02-02T00:00:00Z',
    });
    expect(get).toHaveBeenCalledWith('server/updates', { force: false });
  });

  it('should leave missing versions undefined', async () => {
    jest.mocked(get).mockResolvedValue({});

    expect(await checkForUpdates()).toEqual({
      installed: undefined,
      latest: undefined,
      updateAvailable: false,
      latestNewsUrl: undefined,
      lastChecked: undefined,
    });
  });
});

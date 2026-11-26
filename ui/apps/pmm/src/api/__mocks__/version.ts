import { DistributionMethod, VersionResponse } from 'types/version.types';
import { vi } from 'vitest';
export const VERSION_MOCK: VersionResponse = {
  version: '0.0.0',
  server: {
    version: '0.0.0',
    fullVersion: '0.0.0-00',
    timestamp: '2026-01-01T00:00:00Z',
  },
  managed: {
    version: '0.0.0',
    fullVersion: '2cccd1107b56ff924b34dbe77ebaad2d021c30ea',
    timestamp: '2026-01-01T00:00:00Z',
  },
  distributionMethod: DistributionMethod.unspecified,
};

export const getVersion = vi.fn(
  async (): Promise<VersionResponse> => Promise.resolve(VERSION_MOCK)
);

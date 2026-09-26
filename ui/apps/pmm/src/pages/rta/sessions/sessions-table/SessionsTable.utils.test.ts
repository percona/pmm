import { describe, it, expect } from 'vitest';
import { ServiceType } from 'types/services.types';
import { RealtimeSessionStatus } from 'types/rta.types';
import {
  TEST_REAL_TIME_SESSION,
  TEST_REAL_TIME_SESSION_MYSQL,
} from 'utils/testStubs';
import { getSessionRows } from './SessionsTable.utils';

describe('getSessionRows', () => {
  it('carries the technology onto standalone service rows', () => {
    const [row] = getSessionRows([
      { ...TEST_REAL_TIME_SESSION_MYSQL, clusterName: '' },
    ]);

    expect(row.type).toBe('service');
    expect(row.serviceType).toBe(ServiceType.mysql);
  });

  it('gives a cluster row the technology its services share', () => {
    const [row] = getSessionRows([
      TEST_REAL_TIME_SESSION_MYSQL,
      {
        ...TEST_REAL_TIME_SESSION_MYSQL,
        serviceId: 'service-4',
        serviceName: 'Service 4',
      },
    ]);

    expect(row.type).toBe('cluster');
    expect(row.serviceType).toBe(ServiceType.mysql);
    expect(row.serviceSessions).toHaveLength(2);
  });

  it('leaves a cluster row without a technology when its services disagree', () => {
    const [row] = getSessionRows([
      { ...TEST_REAL_TIME_SESSION_MYSQL, clusterName: 'mixed' },
      {
        ...TEST_REAL_TIME_SESSION,
        clusterName: 'mixed',
        status: RealtimeSessionStatus.running,
      },
    ]);

    expect(row.type).toBe('cluster');
    expect(row.serviceType).toBeUndefined();
  });
});

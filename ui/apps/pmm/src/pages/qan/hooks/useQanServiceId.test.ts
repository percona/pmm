import { describe, expect, it } from 'vitest';
import { resolveServiceUuid } from '../utils/qanServiceResolve';

describe('resolveServiceUuid', () => {
  const services = [
    { serviceId: 'uuid-mysql', serviceName: 'mysql-prod' },
    { serviceId: 'uuid-pg', serviceName: 'pg-analytics' },
  ];

  it('prefers service_id label', () => {
    expect(
      resolveServiceUuid({ service_id: ['/service_id/uuid-mysql'] }, services)
    ).toBe('uuid-mysql');
  });

  it('resolves service_name via inventory', () => {
    expect(resolveServiceUuid({ service_name: ['mysql-prod'] }, services)).toBe('uuid-mysql');
  });

  it('falls back to example service id', () => {
    expect(resolveServiceUuid({}, services, '/service_id/uuid-pg')).toBe('uuid-pg');
  });
});

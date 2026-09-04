import { describe, expect, it } from 'vitest';
import { ServiceType } from 'types/services.types';
import { ServiceOption } from './ServicesAutocompleteInput.types';
import { AvailableService } from 'types/rta.types';
import {
  getServiceOptions,
  isServiceOptionDisabled,
} from './ServicesAutocompleteInput.utils';

const mysqlService: ServiceOption = {
  type: 'service',
  id: 'service-1',
  label: 'MySQL service',
  serviceId: 'service-1',
  cluster: 'cluster-1',
  serviceType: ServiceType.mysql,
};

const mongoService: ServiceOption = {
  ...mysqlService,
  id: 'service-2',
  label: 'MongoDB service',
  serviceId: 'service-2',
  serviceType: ServiceType.mongodb,
};

// sharedTechnology leaves serviceType unset when a cluster's services disagree.
const mixedCluster: ServiceOption = {
  type: 'cluster',
  id: 'cluster-mixed',
  label: 'mixed',
  cluster: 'mixed',
  serviceType: undefined,
};

const mysqlCluster: ServiceOption = {
  ...mixedCluster,
  id: 'cluster-mysql',
  label: 'all-mysql',
  cluster: 'all-mysql',
  serviceType: ServiceType.mysql,
};

describe('isServiceOptionDisabled', () => {
  it('disables nothing when technologies may be mixed', () => {
    for (const option of [mysqlService, mongoService, mixedCluster]) {
      expect(isServiceOptionDisabled(option, undefined, false)).toBe(false);
    }
  });

  it('allows any single-technology option before the technology is fixed', () => {
    expect(isServiceOptionDisabled(mysqlService, undefined, true)).toBe(false);
    expect(isServiceOptionDisabled(mongoService, undefined, true)).toBe(false);
    expect(isServiceOptionDisabled(mysqlCluster, undefined, true)).toBe(false);
  });

  // One click on a mixed cluster would otherwise add both technologies at once,
  // and resolveSelection would later drop half of the user's picks.
  it('disables a mixed cluster even before the technology is fixed', () => {
    expect(isServiceOptionDisabled(mixedCluster, undefined, true)).toBe(true);
  });

  it('disables the other technologies once one is fixed', () => {
    expect(isServiceOptionDisabled(mysqlService, ServiceType.mysql, true)).toBe(
      false
    );
    expect(isServiceOptionDisabled(mongoService, ServiceType.mysql, true)).toBe(
      true
    );
    expect(isServiceOptionDisabled(mysqlCluster, ServiceType.mysql, true)).toBe(
      false
    );
    expect(isServiceOptionDisabled(mixedCluster, ServiceType.mysql, true)).toBe(
      true
    );
  });
});

describe('mixed-technology clusters', () => {
  const mixed = [
    {
      serviceId: 'my-1',
      serviceName: 'mysql-node',
      cluster: 'mixed',
      serviceType: ServiceType.mysql,
    },
    {
      serviceId: 'mo-1',
      serviceName: 'mongo-node',
      cluster: 'mixed',
      serviceType: ServiceType.mongodb,
    },
  ] as unknown as AvailableService[];

  it('leaves the cluster header unselectable in every technology group', () => {
    // The list is grouped by technology, so a cluster spanning two appears under both. Each
    // group looks uniform on its own; the technology has to be judged across the whole
    // cluster or the header becomes selectable and seeds a mixed set in one click.
    const headers = getServiceOptions(mixed).filter(
      (option) => option.type === 'cluster'
    );

    expect(headers).toHaveLength(2);
    headers.forEach((header) => {
      expect(header.serviceType).toBeUndefined();
      expect(isServiceOptionDisabled(header, undefined, true)).toBe(true);
    });
  });

  it('gives each header its own id so they do not collide as keys', () => {
    const ids = getServiceOptions(mixed)
      .filter((option) => option.type === 'cluster')
      .map((option) => option.id);

    expect(new Set(ids).size).toBe(ids.length);
  });

  it('keeps a single-technology cluster header selectable', () => {
    const uniform = [
      {
        serviceId: 'my-1',
        serviceName: 'a',
        cluster: 'prod',
        serviceType: ServiceType.mysql,
      },
      {
        serviceId: 'my-2',
        serviceName: 'b',
        cluster: 'prod',
        serviceType: ServiceType.mysql,
      },
    ] as unknown as AvailableService[];

    const header = getServiceOptions(uniform).find(
      (option) => option.type === 'cluster'
    );

    expect(header?.serviceType).toBe(ServiceType.mysql);
    expect(isServiceOptionDisabled(header!, undefined, true)).toBe(false);
  });
});

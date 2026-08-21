import { describe, expect, it } from 'vitest';
import { ServiceType } from 'types/services.types';
import { ServiceOption } from './ServicesAutocompleteInput.types';
import { isServiceOptionDisabled } from './ServicesAutocompleteInput.utils';

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

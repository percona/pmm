import { describe, expect, it } from 'vitest';
import { ServiceType } from 'types/services.types';
import { ServiceOption } from './ServicesAutocompleteInput.types';
import { AvailableService } from 'types/rta.types';
import {
  getClusterSelectionState,
  getServiceOptions,
  isServiceOptionDisabled,
  toggleClusterServices,
} from './ServicesAutocompleteInput.utils';

const mysqlService: ServiceOption = {
  type: 'service',
  id: 'service-1',
  label: 'MySQL service',
  serviceId: 'service-1',
  cluster: 'cluster-1',
  serviceType: ServiceType.mysql,
  technology: 'MySQL',
};

const mongoService: ServiceOption = {
  ...mysqlService,
  id: 'service-2',
  label: 'MongoDB service',
  serviceId: 'service-2',
  serviceType: ServiceType.mongodb,
  technology: 'MongoDB',
};

// sharedTechnology leaves serviceType unset when a cluster's services disagree.
const mixedCluster: ServiceOption = {
  type: 'cluster',
  id: 'cluster-mixed',
  label: 'mixed',
  cluster: 'mixed',
  serviceType: undefined,
  technology: 'MySQL',
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

  // One click would otherwise add both technologies at once.
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
    // Each technology group looks uniform on its own; only the whole cluster is mixed.
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

  // MUI opens a new group whenever an option's group differs from the previous one.
  it('lists a mixed cluster header in the group it is rendered under', () => {
    const groups = getServiceOptions(mixed).map((option) => option.technology);

    expect(groups).toEqual(['MongoDB', 'MongoDB', 'MySQL', 'MySQL']);
    // Contiguous: one run per technology, so MUI renders each heading once.
    expect(new Set(groups).size).toBe(
      groups.filter((group, index) => group !== groups[index - 1]).length
    );
  });

  // Both headers share the cluster name; acting by name alone would select both
  // technologies at once.
  it('toggles only the services of the header own technology group', () => {
    const options = getServiceOptions(mixed);
    const mysqlHeader = options.find(
      (option) => option.type === 'cluster' && option.technology === 'MySQL'
    )!;

    const selected = toggleClusterServices(mysqlHeader, options, []);

    expect(selected.map((option) => option.serviceId)).toEqual(['my-1']);
  });

  it('reports each header state from its own technology group', () => {
    const options = getServiceOptions(mixed);
    const [mongoHeader, mysqlHeader] = options.filter(
      (option) => option.type === 'cluster'
    );
    const mysqlSelected = options.filter(
      (option) => option.type === 'service' && option.technology === 'MySQL'
    );

    expect(getClusterSelectionState(mysqlHeader, options, mysqlSelected)).toBe(
      'all'
    );
    // Not 'partial', as judging by cluster name alone would make it.
    expect(getClusterSelectionState(mongoHeader, options, mysqlSelected)).toBe(
      'none'
    );
  });

  it('leaves a single-technology cluster toggling its whole cluster', () => {
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
    const options = getServiceOptions(uniform);
    const header = options.find((option) => option.type === 'cluster')!;

    const selected = toggleClusterServices(header, options, []);

    expect(selected.map((option) => option.serviceId)).toEqual([
      'my-1',
      'my-2',
    ]);
    expect(getClusterSelectionState(header, options, selected)).toBe('all');
    expect(toggleClusterServices(header, options, selected)).toEqual([]);
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

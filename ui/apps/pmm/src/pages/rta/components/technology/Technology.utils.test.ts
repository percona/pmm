import { describe, it, expect } from 'vitest';
import { ServiceType } from 'types/services.types';
import { sharedTechnology, technologyLabel } from './Technology.utils';

describe('technologyLabel', () => {
  it('names the technologies RTA supports', () => {
    expect(technologyLabel(ServiceType.mysql)).toBe('MySQL');
    expect(technologyLabel(ServiceType.mongodb)).toBe('MongoDB');
  });

  it('returns an empty label for anything else', () => {
    expect(technologyLabel(undefined)).toBe('');
    expect(technologyLabel(ServiceType.posgresql)).toBe('');
  });
});

describe('sharedTechnology', () => {
  it('returns the common technology', () => {
    expect(sharedTechnology([ServiceType.mysql, ServiceType.mysql])).toBe(
      ServiceType.mysql
    );
  });

  it('returns nothing when the services disagree', () => {
    expect(
      sharedTechnology([ServiceType.mysql, ServiceType.mongodb])
    ).toBeUndefined();
  });
});

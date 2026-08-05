import { describe, it, expect } from 'vitest';
import { ServiceType } from 'types/services.types';
import {
  hasMixedTechnologies,
  sharedTechnology,
  technologyLabel,
} from './Technology.utils';

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

describe('hasMixedTechnologies', () => {
  it('is false for a single technology, however many services', () => {
    expect(hasMixedTechnologies([])).toBe(false);
    expect(hasMixedTechnologies([ServiceType.mongodb])).toBe(false);
    expect(
      hasMixedTechnologies([ServiceType.mongodb, ServiceType.mongodb])
    ).toBe(false);
  });

  it('is true once two technologies are present', () => {
    expect(hasMixedTechnologies([ServiceType.mongodb, ServiceType.mysql])).toBe(
      true
    );
  });

  it('ignores services with no technology', () => {
    expect(hasMixedTechnologies([ServiceType.mysql, undefined])).toBe(false);
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

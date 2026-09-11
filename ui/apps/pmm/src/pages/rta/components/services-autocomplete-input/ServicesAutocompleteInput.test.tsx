import { render } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { ServiceType } from 'types/services.types';
import { ServicesAutocompleteInput } from './index';

const services = [
  {
    serviceId: 'a',
    serviceName: 'alpha',
    serviceType: ServiceType.mysql,
    clusterName: '',
  },
  {
    serviceId: 'b',
    serviceName: 'beta',
    serviceType: ServiceType.mysql,
    clusterName: '',
  },
] as never;

const wrapOf = (tagPresentation: 'label' | 'tags') => {
  const { container } = render(
    <ServicesAutocompleteInput
      tagPresentation={tagPresentation}
      services={services}
      serviceIds={['a', 'b']}
      onServiceIdsChange={() => {}}
    />
  );
  const root = container.querySelector('.MuiAutocomplete-inputRoot')!;

  return getComputedStyle(root).flexWrap;
};

describe('ServicesAutocompleteInput', () => {
  it('keeps the one-line summary on one line', () => {
    expect(wrapOf('label')).toBe('nowrap');
  });

  it('leaves the chip presentation free to wrap', () => {
    // The chip row is meant to wrap; forcing nowrap here squeezed the text input on the
    // selection screen instead.
    expect(wrapOf('tags')).not.toBe('nowrap');
  });
});

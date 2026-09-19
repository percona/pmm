import { fireEvent, render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
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

// Two services in one cluster, so the cluster row toggles both at once.
const clusteredServices = [
  {
    serviceId: 'a',
    serviceName: 'alpha',
    serviceType: ServiceType.mysql,
    clusterName: 'cluster-1',
  },
  {
    serviceId: 'b',
    serviceName: 'beta',
    serviceType: ServiceType.mysql,
    clusterName: 'cluster-1',
  },
] as never;

// One cluster whose services disagree on technology: disabled while a single-technology view
// is being assembled, because picking it would seed a mixed set in one keystroke.
const mixedClusterServices = [
  {
    serviceId: 'm1',
    serviceName: 'mysql-node',
    serviceType: ServiceType.mysql,
    clusterName: 'mixed-cluster',
  },
  {
    serviceId: 'm2',
    serviceName: 'mongo-node',
    serviceType: ServiceType.mongodb,
    clusterName: 'mixed-cluster',
  },
] as never;

const renderPicker = (serviceIds: string[] = []) => {
  const onServiceIdsChange = vi.fn();
  render(
    <ServicesAutocompleteInput
      services={clusteredServices}
      serviceIds={serviceIds}
      onServiceIdsChange={onServiceIdsChange}
    />
  );
  const input = screen.getByRole('combobox');
  fireEvent.mouseDown(input);

  return { input, onServiceIdsChange };
};

// Puts the keyboard highlight on a named row. MUI highlights the option under the pointer,
// which is stable, unlike counting ArrowDown presses: the index the list starts on depends on
// whether anything is already selected.
const highlight = (label: string) => {
  const option = screen
    .getAllByRole('option')
    .find((element) => element.textContent === label);
  if (!option) {
    throw new Error(`no option labelled ${label}`);
  }
  fireEvent.mouseMove(option);
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

  // The cluster toggle lives on the option row's onClick, which the keyboard never reaches:
  // MUI keeps focus on the input and routes Enter through its own selection, where
  // getServiceIds drops cluster options and the selection silently does not change.
  it('toggles a cluster with Enter', () => {
    const { input, onServiceIdsChange } = renderPicker();

    highlight('cluster-1');
    fireEvent.keyDown(input, { key: 'Enter' });

    expect(onServiceIdsChange).toHaveBeenCalledWith(['a', 'b']);
  });

  it('toggles a cluster with Space', () => {
    // Space does not select in MUI at all - it types into the input - so without this the
    // cluster row has no Space behaviour to correct, only one to add.
    const { input, onServiceIdsChange } = renderPicker();

    highlight('cluster-1');
    fireEvent.keyDown(input, { key: ' ' });

    expect(onServiceIdsChange).toHaveBeenCalledWith(['a', 'b']);
  });

  it('clears a fully selected cluster with Enter', () => {
    const { input, onServiceIdsChange } = renderPicker(['a', 'b']);

    highlight('cluster-1');
    fireEvent.keyDown(input, { key: 'Enter' });

    expect(onServiceIdsChange).toHaveBeenCalledWith([]);
  });

  it('does not hijack Enter on a plain service row', () => {
    // The interception is scoped to cluster rows. With a service highlighted the handler must
    // stand aside and leave the key to MUI, which here selects nothing because its Enter uses
    // the index its own keyboard navigation set, not the one a hover produced. What matters is
    // that the cluster toggle did not fire: that would select the whole cluster from a row the
    // user was not on.
    const { input, onServiceIdsChange } = renderPicker();

    highlight('alpha');
    fireEvent.keyDown(input, { key: 'Enter' });

    expect(onServiceIdsChange).not.toHaveBeenCalled();
  });

  it('does not toggle a disabled cluster', () => {
    // A cluster spanning technologies is disabled while one view is being assembled, and the
    // keyboard must respect that the same way the mouse path does.
    const onServiceIdsChange = vi.fn();
    render(
      <ServicesAutocompleteInput
        singleTechnology
        services={mixedClusterServices}
        serviceIds={[]}
        onServiceIdsChange={onServiceIdsChange}
      />
    );
    const input = screen.getByRole('combobox');
    fireEvent.mouseDown(input);

    highlight('mixed-cluster');
    fireEvent.keyDown(input, { key: 'Enter' });

    expect(onServiceIdsChange).not.toHaveBeenCalled();
  });
});

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

// A cluster whose services disagree on technology, disabled in a single-technology view.
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

// Hovers rather than counting ArrowDown presses, whose start index depends on the selection.
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
    // Forcing nowrap on the chip row would squeeze the text input.
    expect(wrapOf('tags')).not.toBe('nowrap');
  });

  // The keyboard never reaches the row's onClick, and MUI's own Enter drops cluster options.
  it('toggles a cluster with Enter', () => {
    const { input, onServiceIdsChange } = renderPicker();

    highlight('cluster-1');
    fireEvent.keyDown(input, { key: 'Enter' });

    expect(onServiceIdsChange).toHaveBeenCalledWith(['a', 'b']);
  });

  it('toggles a cluster with Space', () => {
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
    // MUI's own Enter may select the hovered row; only the cluster toggle would add its sibling.
    const { input, onServiceIdsChange } = renderPicker();

    highlight('alpha');
    fireEvent.keyDown(input, { key: 'Enter' });

    expect(onServiceIdsChange).not.toHaveBeenCalledWith(['a', 'b']);
  });

  it('does not toggle a disabled cluster', () => {
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

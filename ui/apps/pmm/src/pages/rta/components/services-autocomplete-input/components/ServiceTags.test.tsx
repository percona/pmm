import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { ServiceOption } from '../ServicesAutocompleteInput.types';
import ServiceTags from './ServiceTags';

const option = (label: string): ServiceOption =>
  ({ id: label, label, type: 'service' }) as ServiceOption;

const getTagProps = vi.fn(() => ({ key: 'k', onDelete: vi.fn() })) as never;

const renderTags = (labels: string[]) =>
  render(
    <ServiceTags
      tagPresentation="label"
      value={labels.map(option)}
      getTagProps={getTagProps}
    />
  );

describe('ServiceTags label presentation', () => {
  it('names a single selected service outright', () => {
    renderTags(['rta-mysql-blocking']);

    expect(screen.getByText('rta-mysql-blocking')).toBeInTheDocument();
    expect(screen.queryByText(/^\+/)).not.toBeInTheDocument();
  });

  it('names the first service and counts the rest instead of joining every name', () => {
    // Joining them produced a string wider than the field, which pushed the autocomplete's
    // input and clear button onto a second line and grew the control into the toolbar.
    renderTags(['rta-mysql-blocking', 'rta-mysql57-degrade', 'rta-nolocks']);

    expect(screen.getByText('rta-mysql-blocking')).toBeInTheDocument();
    expect(screen.getByText('+2')).toBeInTheDocument();
    expect(
      screen.queryByText(/rta-mysql-blocking, rta-mysql57-degrade/)
    ).not.toBeInTheDocument();
  });

  it('keeps the full selection reachable as a tooltip', () => {
    renderTags(['alpha', 'beta', 'gamma']);

    expect(screen.getByLabelText('alpha, beta, gamma')).toBeInTheDocument();
  });

  it('renders nothing rather than throwing on an empty selection', () => {
    expect(() => renderTags([])).not.toThrow();
  });
});

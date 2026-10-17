import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { TestWrapper } from 'utils/testWrapper';
import { LogsFilterBar } from './LogsFilterBar';

const filter = {
  start: '2026-10-17T10:00:00.000Z',
  end: '2026-10-17T11:00:00.000Z',
};

const facets = {
  nodeNames: [
    { value: 'pg-1', count: 12 },
    { value: 'mongo-1', count: 3 },
  ],
  sources: [],
  presets: [],
};

const renderBar = (
  props: Partial<Parameters<typeof LogsFilterBar>[0]> = {}
) => {
  const onChange = vi.fn();
  render(
    <TestWrapper>
      <LogsFilterBar
        filter={filter}
        range="1h"
        facets={facets}
        onChange={onChange}
        onRefresh={vi.fn()}
        {...props}
      />
    </TestWrapper>
  );
  return onChange;
};

describe('LogsFilterBar', () => {
  it('shows each filter value inside its button', () => {
    renderBar({ filter: { ...filter, nodeNames: ['pg-1'], minSeverity: 13 } });
    expect(screen.getByRole('button', { name: 'Node pg-1' })).toBeVisible();
    expect(screen.getByRole('button', { name: 'Source Any' })).toBeVisible();
    expect(
      screen.getByRole('button', { name: 'Severity Warning and above' })
    ).toBeVisible();
  });

  it('switches to a quick range', () => {
    const onChange = renderBar();
    fireEvent.click(screen.getByRole('button', { name: 'Last 6 hours' }));
    expect(onChange).toHaveBeenCalledWith(filter, '6h');
  });

  it('filters by a node picked from the Node button', () => {
    const onChange = renderBar();
    fireEvent.click(screen.getByRole('button', { name: 'Node Any' }));
    fireEvent.click(screen.getByRole('option', { name: 'pg-1 (12)' }));
    expect(onChange).toHaveBeenCalledWith(
      { ...filter, nodeNames: ['pg-1'] },
      '1h'
    );
  });

  it('sets the minimum severity from the Severity button', () => {
    const onChange = renderBar();
    fireEvent.click(screen.getByRole('button', { name: 'Severity Any' }));
    fireEvent.click(screen.getByRole('menuitem', { name: 'Error and above' }));
    expect(onChange).toHaveBeenCalledWith({ ...filter, minSeverity: 17 }, '1h');
  });

  it('edits the start of the range from the time button', () => {
    const onChange = renderBar();
    fireEvent.click(screen.getByTestId('logs-time-button'));
    fireEvent.change(screen.getByLabelText('From'), {
      target: { value: '2026-10-17T09:30' },
    });
    expect(onChange).toHaveBeenCalledWith({
      ...filter,
      start: new Date('2026-10-17T09:30').toISOString(),
    });
  });

  it('clears every filter but the time range', () => {
    const onChange = renderBar({
      filter: {
        ...filter,
        nodeNames: ['pg-1'],
        presets: ['postgres'],
        minSeverity: 13,
        text: 'lock',
        pattern: 'deadlock detected',
      },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Clear filters' }));
    expect(onChange).toHaveBeenCalledWith(filter, '1h');
  });

  it('offers to clear filters only when one is set', () => {
    renderBar();
    expect(
      screen.queryByRole('button', { name: 'Clear filters' })
    ).not.toBeInTheDocument();
  });

  it('searches once typing stops', async () => {
    const onChange = renderBar();
    fireEvent.change(screen.getByRole('textbox', { name: 'Search' }), {
      target: { value: 'lock' },
    });
    await waitFor(() =>
      expect(onChange).toHaveBeenCalledWith({ ...filter, text: 'lock' }, '1h')
    );
  });
});

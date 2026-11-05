import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { DataFrame, LoadingState, PanelProps } from '@grafana/data';
import { locationService } from '@grafana/runtime';
import { AuroraTreePanel } from './AuroraTreePanel';
import { DEFAULT_OPTIONS, SEARCH_DEBOUNCE_MS } from './AuroraTreePanel.constants';
import { mergedFrame, timeRange } from './AuroraTreePanel.mocks';
import { AuroraTreeOptions } from './AuroraTreePanel.types';

const mockSearch: { current: Record<string, string | string[]> } = { current: {} };

jest.mock('@grafana/runtime', () => ({
  config: { appSubUrl: '/graph' },
  locationService: {
    getSearchObject: () => mockSearch.current,
    partial: jest.fn(),
  },
}));

const renderPanel = (series: DataFrame[]) => {
  const props = {
    data: { series, state: LoadingState.Done, timeRange },
    options: DEFAULT_OPTIONS,
    timeRange,
  } as unknown as PanelProps<AuroraTreeOptions>;

  return render(<AuroraTreePanel {...props} />);
};

const typeSearch = (text: string) => fireEvent.change(
  screen.getByPlaceholderText('Search clusters, instances, accounts'),
  { target: { value: text } },
);

describe('AuroraTreePanel::', () => {
  beforeEach(() => {
    jest.useFakeTimers();
    mockSearch.current = {};
    jest.mocked(locationService.partial).mockClear();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('renders clusters collapsed with their aggregates', () => {
    renderPanel([mergedFrame]);

    expect(screen.getByText('database-3')).toBeInTheDocument();
    expect(screen.getByText('AWS cluster unknown')).toBeInTheDocument();
    expect(screen.getAllByText('1 writer')).toHaveLength(3);
    expect(screen.getByText('changing')).toBeInTheDocument();
    expect(screen.getByText('24.75 qps')).toBeInTheDocument();
    expect(screen.queryByRole('link')).not.toBeInTheDocument();
  });

  it('links every service to its details dashboard once expanded', () => {
    renderPanel([mergedFrame]);

    fireEvent.click(screen.getByRole('button', { name: 'Expand all rows' }));

    expect(screen.getAllByRole('link')).toHaveLength(15);
    expect(screen.getByRole('link', { name: 'database-3-instance-1-reader' })).toHaveAttribute(
      'href',
      '/graph/d/mysql-amazonaurora/mysql-amazon-aurora-details?var-service_name=database-3-instance-1-reader',
    );
    expect(screen.getByRole('link', { name: 'MySQL Instance Summary for database-3-instance-1-reader' })).toHaveAttribute(
      'href',
      '/graph/d/mysql-instance-summary/mysql-instance-summary?var-service_name=database-3-instance-1-reader',
    );
    expect(screen.getByRole('link', { name: 'Node Summary for database-3-instance-1-reader' })).toHaveAttribute(
      'href',
      '/graph/d/node-instance-summary/node-summary?var-node_name=database-3-instance-1-reader',
    );
    expect(screen.getByRole('link', { name: 'Node Summary for aurora-stale' })).toHaveAttribute(
      'href',
      '/graph/d/node-instance-summary/node-summary?var-node_name=aurora-stale',
    );
    expect(screen.getByText('reader')).toBeInTheDocument();
    expect(screen.getByText('unknown')).toBeInTheDocument();
    expect(screen.getByText('Down')).toBeInTheDocument();
  });

  it('sorts clusters by a column, missing values last', () => {
    renderPanel([mergedFrame]);

    fireEvent.click(screen.getByRole('button', { name: 'Queries/s' }));
    fireEvent.click(screen.getByRole('button', { name: 'Queries/s' }));

    const rows = screen.getAllByRole('row').slice(1).map((row) => row.textContent);

    expect(rows[0]).toContain('database-3');
    expect(rows[3]).toContain('AWS cluster unknown');
  });

  it('renders the toolbar with the cluster count', () => {
    renderPanel([mergedFrame]);

    expect(screen.getByPlaceholderText('Search clusters, instances, accounts')).toBeInTheDocument();
    expect(screen.getByLabelText('Region')).toBeInTheDocument();
    expect(screen.getByLabelText('AWS account')).toBeInTheDocument();
    expect(screen.getByLabelText('Cluster')).toBeInTheDocument();
    expect(screen.getByText('4 of 4 clusters')).toBeInTheDocument();
    expect(locationService.partial).not.toHaveBeenCalled();
  });

  it('searches after typing stops, expands clusters matched by instance and writes the URL', async () => {
    renderPanel([mergedFrame]);
    typeSearch('reader');
    jest.advanceTimersByTime(SEARCH_DEBOUNCE_MS - 50);

    expect(screen.getByText('4 of 4 clusters')).toBeInTheDocument();
    expect(await screen.findByText('1 of 4 clusters')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'database-3-instance-1-reader' })).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'database-3-instance-1' })).not.toBeInTheDocument();
    expect(locationService.partial).toHaveBeenLastCalledWith(
      {
        'aurora-q': 'reader', 'aurora-region': null, 'aurora-account': null, 'aurora-cluster': null,
      },
      true,
    );
  });

  it('restores the region filter from the URL', () => {
    mockSearch.current = { 'aurora-region': 'eu-north-1' };
    renderPanel([mergedFrame]);

    expect(screen.getByText('3 of 4 clusters')).toBeInTheDocument();
    expect(screen.getByText('AWS cluster unknown')).toBeInTheDocument();
    expect(screen.queryByText('pmm-qa-aurora3-mysql')).not.toBeInTheDocument();
  });

  it('restores filters from the URL', () => {
    mockSearch.current = { 'aurora-account': '222222222222' };
    renderPanel([mergedFrame]);

    expect(screen.getByText('1 of 4 clusters')).toBeInTheDocument();
    expect(screen.getByText('pmm-qa-aurora3-mysql')).toBeInTheDocument();
    expect(screen.queryByText('database-3')).not.toBeInTheDocument();
  });

  it('shows a no-match state that clears the filters', async () => {
    renderPanel([mergedFrame]);
    typeSearch('no-such-cluster');

    expect(await screen.findByText('No clusters match the filters')).toBeInTheDocument();
    expect(screen.getByText('0 of 4 clusters')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Clear filters' }));

    expect(screen.getByText('4 of 4 clusters')).toBeInTheDocument();
  });

  it('renders an empty state without data', () => {
    renderPanel([]);

    expect(screen.getByText('No Amazon Aurora MySQL instances found')).toBeInTheDocument();
  });
});

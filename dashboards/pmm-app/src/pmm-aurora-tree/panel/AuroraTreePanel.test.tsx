import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { DataFrame, LoadingState, PanelProps } from '@grafana/data';
import { locationService } from '@grafana/runtime';
import { AuroraTreePanel } from './AuroraTreePanel';
import { DEFAULT_OPTIONS, SEARCH_DEBOUNCE_MS } from './AuroraTreePanel.constants';
import { queryFrames, timeRange } from './AuroraTreePanel.mocks';
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
  screen.getByPlaceholderText('Search groups, instances, accounts'),
  { target: { value: text } },
);

const groupRows = () => screen.getAllByRole('row').slice(1).map((row) => row.textContent);

describe('AuroraTreePanel::', () => {
  beforeEach(() => {
    jest.useFakeTimers();
    mockSearch.current = {};
    jest.mocked(locationService.partial).mockClear();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('renders groups collapsed with their engine, kind and aggregates', () => {
    renderPanel(queryFrames);

    expect(screen.getByText('database-3')).toBeInTheDocument();
    expect(screen.getAllByText('AWS info unavailable')).toHaveLength(2);
    expect(screen.getAllByText('cluster')).toHaveLength(3);
    expect(screen.getByText('replication group')).toBeInTheDocument();
    expect(screen.getAllByText('instance')).toHaveLength(2);
    expect(screen.getByText('Aurora PostgreSQL')).toBeInTheDocument();
    expect(screen.getByText('MariaDB')).toBeInTheDocument();
    expect(screen.getAllByText('1 writer')).toHaveLength(7);
    expect(screen.getByText('unknown')).toBeInTheDocument();
    expect(screen.getByText('24.75 qps')).toBeInTheDocument();
    expect(screen.getByText('17.00 tps')).toBeInTheDocument();
    expect(screen.getByText('18 GiB free (min)')).toBeInTheDocument();
    expect(screen.getByText('50 GiB free')).toBeInTheDocument();
    expect(screen.queryByRole('link')).not.toBeInTheDocument();
  });

  it('links every instance to the dashboards of its engine once expanded', () => {
    renderPanel(queryFrames);

    fireEvent.click(screen.getByRole('button', { name: 'Expand all rows' }));

    expect(screen.getAllByRole('link')).toHaveLength(27);
    expect(screen.getByRole('link', { name: 'database-3-instance-1-reader' })).toHaveAttribute(
      'href',
      '/graph/d/mysql-amazonaurora/mysql-amazon-aurora-details?var-service_name=database-3-instance-1-reader',
    );
    expect(screen.getByRole('link', { name: 'MySQL Instance Summary for database-3-instance-1-reader' }))
      .toHaveAttribute(
        'href',
        '/graph/d/mysql-instance-summary/mysql-instance-summary?var-service_name=database-3-instance-1-reader',
      );
    expect(screen.getByRole('link', { name: 'rds-mysql-1-replica' })).toHaveAttribute(
      'href',
      '/graph/d/mysql-instance-summary/mysql-instance-summary?var-service_name=rds-mysql-1-replica',
    );
    expect(screen.getByRole('link', { name: 'database-4-instance-1' })).toHaveAttribute(
      'href',
      '/graph/d/postgresql-instance-summary/postgresql-instance-summary?var-service_name=database-4-instance-1',
    );
    expect(screen.getByRole('link', { name: 'Node Summary for aurora-stale' })).toHaveAttribute(
      'href',
      '/graph/d/node-instance-summary/node-summary?var-node_name=aurora-stale',
    );
    screen.getAllByRole('link').forEach((link) => {
      expect(link).toHaveAttribute('target', '_blank');
      expect(link).toHaveAttribute('rel', 'noopener noreferrer');
    });
    expect(screen.getByText('rds-mysql-1-replica-west, us-west-2')).toBeInTheDocument();
    expect(screen.getAllByText('reader')).toHaveLength(4);
    expect(screen.getByText('Down')).toBeInTheDocument();
  });

  it('sorts groups by a column, missing values last', () => {
    renderPanel(queryFrames);

    fireEvent.click(screen.getByRole('button', { name: 'Throughput' }));
    fireEvent.click(screen.getByRole('button', { name: 'Throughput' }));

    const rows = groupRows();

    expect(rows[0]).toContain('rds-mysql-1');
    expect(rows[7]).toContain('AWS info unavailable');
  });

  it('renders the toolbar with the group count', () => {
    renderPanel(queryFrames);

    expect(screen.getByPlaceholderText('Search groups, instances, accounts')).toBeInTheDocument();
    expect(screen.getByLabelText('Region')).toBeInTheDocument();
    expect(screen.getByLabelText('AWS account')).toBeInTheDocument();
    expect(screen.getByLabelText('Engine')).toBeInTheDocument();
    expect(screen.getByLabelText('Group')).toBeInTheDocument();
    expect(screen.getByText('8 of 8 groups')).toBeInTheDocument();
    expect(locationService.partial).not.toHaveBeenCalled();
  });

  it('searches after typing stops, expands groups matched by instance and writes the URL', async () => {
    renderPanel(queryFrames);
    typeSearch('reader');
    jest.advanceTimersByTime(SEARCH_DEBOUNCE_MS - 50);

    expect(screen.getByText('8 of 8 groups')).toBeInTheDocument();
    expect(await screen.findByText('2 of 8 groups')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'database-3-instance-1-reader' })).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'database-3-instance-1' })).not.toBeInTheDocument();
    expect(locationService.partial).toHaveBeenLastCalledWith(
      {
        'rds-q': 'reader', 'rds-region': null, 'rds-account': null, 'rds-engine': null, 'rds-group': null,
      },
      true,
    );
  });

  it('restores the engine filter from the URL', () => {
    mockSearch.current = { 'rds-engine': ['postgres', 'aurora-postgresql'] };
    renderPanel(queryFrames);

    expect(screen.getByText('3 of 8 groups')).toBeInTheDocument();
    expect(screen.getByText('database-4')).toBeInTheDocument();
    expect(screen.getByText('rds-pg-1')).toBeInTheDocument();
    expect(screen.queryByText('database-3')).not.toBeInTheDocument();
  });

  it('restores the account filter from the URL', () => {
    mockSearch.current = { 'rds-account': '222222222222' };
    renderPanel(queryFrames);

    expect(screen.getByText('3 of 8 groups')).toBeInTheDocument();
    expect(screen.getByText('rds-mysql-1')).toBeInTheDocument();
    expect(screen.queryByText('database-3')).not.toBeInTheDocument();
  });

  it('shows a no-match state that clears the filters', async () => {
    renderPanel(queryFrames);
    typeSearch('no-such-group');

    expect(await screen.findByText('No groups match the filters')).toBeInTheDocument();
    expect(screen.getByText('0 of 8 groups')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Clear filters' }));

    expect(screen.getByText('8 of 8 groups')).toBeInTheDocument();
  });

  it('renders an empty state without data', () => {
    renderPanel([]);

    expect(screen.getByText('No Amazon RDS instances found')).toBeInTheDocument();
  });
});

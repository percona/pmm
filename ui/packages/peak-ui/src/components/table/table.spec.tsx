import { render, screen } from '@testing-library/react';
import { type MRT_ColumnDef } from 'material-react-table';
import { describe, expect, it } from 'vitest';
import { baseThemeOptions, ThemeContextProvider } from '../../design';
import EmptyState from '../empty-state';
import Table from './table';

type Row = { name: string };

const columns: MRT_ColumnDef<Row>[] = [{ header: 'Name', accessorKey: 'name' }];

const renderEmptyTable = (props: Partial<React.ComponentProps<typeof Table<Row>>> = {}) =>
  render(
    <ThemeContextProvider themeOptions={baseThemeOptions}>
      <Table<Row> tableName="empty-table-test" columns={columns} data={[]} {...props} />
    </ThemeContextProvider>
  );

describe('Table with no data', () => {
  it('renders an EmptyState titled with noDataMessage by default', () => {
    renderEmptyTable({ noDataMessage: 'No servers yet' });

    expect(screen.getByText('No servers yet')).toBeInTheDocument();
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    expect(screen.queryByRole('heading')).not.toBeInTheDocument();
  });

  it('keeps the info alert for tables that opt into noDataVariant="alert"', () => {
    renderEmptyTable({ noDataMessage: 'No servers yet', noDataVariant: 'alert' });

    expect(screen.getByRole('alert')).toHaveTextContent('No servers yet');
  });

  it('renders the emptyState slot instead of the default', () => {
    renderEmptyTable({
      noDataMessage: 'No servers yet',
      emptyState: <EmptyState size="small" title="Nothing here" description="Add a server." />,
    });

    expect(screen.getByText('Nothing here')).toBeInTheDocument();
    expect(screen.getByText('Add a server.')).toBeInTheDocument();
    expect(screen.queryByText('No servers yet')).not.toBeInTheDocument();
  });
});

describe('Table with every row filtered out', () => {
  const filteredOutProps = {
    data: [{ name: 'alpha' }],
    enableGlobalFilter: true,
    initialState: { showGlobalFilter: true, globalFilter: 'zzzzz' },
    emptyFilterResultsMessage: 'No servers match these filters',
  };

  it('renders an EmptyState titled with emptyFilterResultsMessage by default', () => {
    renderEmptyTable(filteredOutProps);

    expect(screen.getByText('No servers match these filters')).toBeInTheDocument();
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('keeps the info alert for tables that opt into noDataVariant="alert"', () => {
    renderEmptyTable({ ...filteredOutProps, noDataVariant: 'alert' });

    expect(screen.getByRole('alert')).toHaveTextContent('No servers match these filters');
  });
});

import { getAllByRole, render, screen } from '@testing-library/react';
import { Table } from './Table';
import { Column } from './Table.types';
import { Messages } from './Table.messages';

const ROWS = [
  {
    id: 1,
    name: 'Name #1',
  },
  {
    id: 2,
    name: 'Name #2',
  },
  {
    id: 3,
    name: 'Name #3',
  },
];

const COLUMNS: Column<(typeof ROWS)[0]>[] = [
  {
    field: 'id',
    name: 'ID',
  },
  {
    field: 'name',
    name: 'Name',
    cell: (item) => <strong data-testid={item.name}>{item.name}</strong>,
  },
];

describe('Table', () => {
  it('shows default empty message', () => {
    render(<Table rowId="id" rows={[]} columns={COLUMNS} />);

    expect(screen.getByTestId('table-empty-message')).toHaveTextContent(
      Messages.noData
    );
  });

  it('shows correct empty message', () => {
    render(
      <Table
        rowId="id"
        rows={[]}
        columns={COLUMNS}
        emptyMessage="Nothing to see here"
      />
    );

    expect(screen.getByTestId('table-empty-message')).toHaveTextContent(
      'Nothing to see here'
    );
  });

  it('shows loading', () => {
    render(<Table rowId="id" isLoading rows={[]} columns={COLUMNS} />);

    expect(screen.getByTestId('table-loading-indicator')).toBeDefined();
  });

  it('shows data', () => {
    render(<Table rowId="id" rows={ROWS} columns={COLUMNS} />);

    const tableBody = screen.getByTestId('table-body');

    expect(getAllByRole(tableBody, 'row')).toHaveLength(ROWS.length);
  });

  it('renders custom cell', () => {
    render(<Table rowId="id" rows={ROWS} columns={COLUMNS} />);

    expect(screen.getByTestId('Name #1')).toBeDefined();
  });
});

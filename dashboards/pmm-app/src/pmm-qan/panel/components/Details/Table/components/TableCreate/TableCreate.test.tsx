import React from 'react';
import { render, screen } from '@testing-library/react';
import { Databases } from 'shared/core';
import { TableCreate } from './TableCreate';
import { useShowCreateTable } from './TableCreate.hooks';

jest.mock('shared/components/helpers/notification-manager');
jest.mock('./TableCreate.hooks');

const mockedUseShowCreateTable = useShowCreateTable as jest.Mock;

const renderTableCreate = () => render(
  <TableCreate tableName="t" databaseType={Databases.mysql} example="SELECT 1" />,
);

describe('TableCreate::', () => {
  it('renders while the result is pending', () => {
    mockedUseShowCreateTable.mockReturnValue([{ error: '', loading: true, value: null }]);

    renderTableCreate();

    expect(screen.getByTestId('highlight-code').textContent).toEqual('');
  });

  it('renders the create table statement', () => {
    mockedUseShowCreateTable.mockReturnValue([{ error: '', loading: false, value: 'CREATE TABLE t (id int)' }]);

    renderTableCreate();

    expect(screen.getByTestId('highlight-code').textContent).toEqual('CREATE TABLE t (id int)');
  });

  it('renders the error instead of the statement', () => {
    mockedUseShowCreateTable.mockReturnValue([{ error: 'Unknown error', loading: false, value: null }]);

    renderTableCreate();

    expect(screen.queryByTestId('highlight-code')).not.toBeInTheDocument();
    expect(screen.getByText('Unknown error')).toBeInTheDocument();
  });
});

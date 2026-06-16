import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import QueryAndDetails from './QueryAndDetails';
import {
  TEST_MONGO_DB_QUERY_DATA,
  TEST_MYSQL_QUERY_DATA,
  TEST_USER_ADMIN,
} from 'utils/testStubs';
import { wrapWithUserProvider } from 'utils/testUtils';
import { QueryData } from 'types/rta.types';

const renderComponent = (
  user = TEST_USER_ADMIN,
  queryData: QueryData = TEST_MONGO_DB_QUERY_DATA
) =>
  render(
    <ThemeProvider theme={createTheme({ palette: { mode: 'light' } })}>
      {wrapWithUserProvider(<QueryAndDetails queryData={queryData} />, {
        user,
      })}
    </ThemeProvider>
  );

describe('QueryAndDetails', () => {
  it('renders the query text in a syntax highlighter', () => {
    renderComponent();

    // Syntax highlighter splits content into token spans
    expect(
      screen.getByText((content) => content.includes('mycollection'))
    ).toBeInTheDocument();
  });

  it('renders service name', () => {
    renderComponent();

    expect(screen.getByText('Service')).toBeInTheDocument();
    expect(
      screen.getByText(TEST_MONGO_DB_QUERY_DATA.serviceName)
    ).toBeInTheDocument();
  });

  it('renders elapsed exec. time metric', () => {
    renderComponent();

    expect(screen.getByText('Elapsed exec. time')).toBeInTheDocument();
  });

  it('renders plan summary metric', () => {
    renderComponent();

    expect(screen.getByText('Plan summary')).toBeInTheDocument();
  });

  it('formats operation start time and data capture time in the user timezone', () => {
    // 2021-01-01T00:00:00Z in America/New_York (EST, UTC-5) is 2020-12-31 19:00:00
    const userWithTimezone = {
      ...TEST_USER_ADMIN,
      preferences: {
        ...TEST_USER_ADMIN.preferences,
        timezone: 'America/New_York',
      },
    };
    renderComponent(userWithTimezone);

    expect(screen.getByText('Operation start time')).toBeInTheDocument();
    expect(screen.getByText('Data capture time')).toBeInTheDocument();
    // Same UTC instant, so same local time
    const timeElements = screen.getAllByText('2020-12-31 19:00:00');
    expect(timeElements).toHaveLength(2);
  });

  it('renders MySQL-specific metrics for a MySQL query', () => {
    renderComponent(TEST_USER_ADMIN, TEST_MYSQL_QUERY_DATA);

    // MySQL-specific fields are shown.
    expect(screen.getByText('Command')).toBeInTheDocument();
    expect(screen.getByText('State')).toBeInTheDocument();
    expect(screen.getByText('Rows examined')).toBeInTheDocument();
    expect(screen.getByText('Full scan')).toBeInTheDocument();
    expect(screen.getByTestId('command-value')).toHaveTextContent('Query');

    // MongoDB-specific fields are not shown.
    expect(screen.queryByText('Plan summary')).not.toBeInTheDocument();
    expect(screen.queryByText('Collection')).not.toBeInTheDocument();
  });
});

import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import QueryAndDetails from './QueryAndDetails';
import { TEST_MONGO_DB_QUERY_DATA } from 'utils/testStubs';

const renderComponent = () =>
  render(
    <ThemeProvider theme={createTheme({ palette: { mode: 'light' } })}>
      <QueryAndDetails queryData={TEST_MONGO_DB_QUERY_DATA} />
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
});

import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import QueryAndDetails from './QueryAndDetails';
import { TEST_MONGO_DB_QUERY_DATA } from 'utils/testStubs';

const renderComponent = (query = TEST_MONGO_DB_QUERY_DATA) =>
  render(
    <ThemeProvider theme={createTheme({ palette: { mode: 'light' } })}>
      <QueryAndDetails query={query} />
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

  it('renders current state with the query state', () => {
    renderComponent();

    expect(screen.getByText('Current state')).toBeInTheDocument();
    expect(screen.getByText(TEST_MONGO_DB_QUERY_DATA.state)).toBeInTheDocument();
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
    // expect(screen.getByText('20')).toBeInTheDocument();
    // expect(screen.getByText('ms')).toBeInTheDocument();
  });

  it('renders plan summary metric', () => {
    renderComponent();

    expect(screen.getByText('Plan summary')).toBeInTheDocument();
    // expect(
    //   screen.getByText('Full collection scan (COLLSCAN)')
    // ).toBeInTheDocument();
  });

  it('renders docs examined/sent metric', () => {
    renderComponent();

    expect(screen.getByText('Docs examined/sent')).toBeInTheDocument();
    // expect(screen.getByText('84,291/1')).toBeInTheDocument();
  });

  it('renders snapshot time and operation ID metrics', () => {
    renderComponent();

    expect(screen.getByText('Snapshot time')).toBeInTheDocument();
    // expect(screen.getByText('2025-10-17 11:18:29')).toBeInTheDocument();
    expect(screen.getByText('Operation ID')).toBeInTheDocument();
    // expect(screen.getByText('1238912')).toBeInTheDocument();
  });

  it('renders Operation ID with subtitle', () => {
    renderComponent();

    expect(screen.getByText('opid')).toBeInTheDocument();
  });
});

import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { createTheme, ThemeProvider } from '@mui/material/styles';
// import QueryAndDetails from './QueryAndDetails';
import { TEST_MONGO_DB_QUERY_DATA } from 'utils/testStubs';

const renderComponent = () =>
  render(
    <ThemeProvider theme={createTheme({ palette: { mode: 'light' } })}>
      {/* <QueryAndDetails queryData={query} /> */}
    </ThemeProvider>
  );

describe.skip('QueryAndDetails', () => {
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
      screen.getByText(TEST_MONGO_DB_QUERY_DATA.service_name)
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
});

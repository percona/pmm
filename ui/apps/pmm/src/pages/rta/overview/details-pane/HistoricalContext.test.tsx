import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import HistoricalContext from './HistoricalContext';

const renderComponent = () =>
  render(
    <ThemeProvider theme={createTheme({ palette: { mode: 'light' } })}>
      <HistoricalContext />
    </ThemeProvider>
  );

describe('HistoricalContext', () => {
  it('renders the section heading', () => {
    renderComponent();

    expect(screen.getByText('Historical Context')).toBeInTheDocument();
  });

  it('renders max exec. time metric', () => {
    renderComponent();

    expect(screen.getByText('Max. exec. time')).toBeInTheDocument();
    // expect(screen.getByText('2421.83')).toBeInTheDocument();
    // expect(screen.getByText('5.01%')).toBeInTheDocument();
    expect(screen.getAllByText('ms').length).toBeGreaterThan(0);
  });

  it('renders average exec. time metric', () => {
    renderComponent();

    expect(screen.getByText('Average exec. time')).toBeInTheDocument();
    // expect(screen.getByText('638.19')).toBeInTheDocument();
  });

  it('renders exec. count metric', () => {
    renderComponent();

    expect(screen.getByText('Exec. count')).toBeInTheDocument();
    // expect(screen.getByText('178')).toBeInTheDocument();
    // expect(screen.getByText('x')).toBeInTheDocument();
  });

  it('renders total exec. time metric', () => {
    renderComponent();

    expect(screen.getByText('Total exec. time')).toBeInTheDocument();
    // expect(screen.getByText('2')).toBeInTheDocument();
    // expect(screen.getByText('min')).toBeInTheDocument();
    // expect(screen.getByText('12.72%')).toBeInTheDocument();
  });
});

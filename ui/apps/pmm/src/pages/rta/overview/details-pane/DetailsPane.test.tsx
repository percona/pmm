import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import DetailsPane from './DetailsPane';
import { Messages } from './DetailsPane.messages';
import { TEST_MONGO_DB_QUERY_DATA } from 'utils/testStubs';

const defaultProps = {
  isFirstQuery: false,
  isLastQuery: false,
  onClose: vi.fn(),
  onNext: vi.fn(),
  onPrevious: vi.fn(),
};

const renderComponent = () =>
  render(
    <ThemeProvider theme={createTheme({ palette: { mode: 'light' } })}>
      <DetailsPane {...defaultProps} query={TEST_MONGO_DB_QUERY_DATA} />
    </ThemeProvider>
  );

describe.skip('DetailsPane', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the pane with aria-hidden when no query', () => {
    renderComponent();

    const pane = screen.getByTestId('query-details-pane');
    expect(pane).toBeInTheDocument();
    expect(pane).toHaveAttribute('aria-hidden', 'true');
  });

  it('renders the pane with aria-hidden false when query is provided', () => {
    renderComponent();

    const pane = screen.getByTestId('query-details-pane');
    expect(pane).toHaveAttribute('aria-hidden', 'false');
  });

  it('renders Details and Raw data tabs', () => {
    renderComponent();

    expect(screen.getByTestId('details-pane-details-tab')).toHaveTextContent(
      Messages.tabs.details
    );
    expect(screen.getByTestId('details-pane-raw-data-tab')).toHaveTextContent(
      Messages.tabs.rawData
    );
  });

  it('calls onClose when close button is clicked', () => {
    renderComponent();

    fireEvent.click(screen.getByTestId('details-pane-close-button'));

    expect(defaultProps.onClose).toHaveBeenCalledTimes(1);
  });

  it('calls onNext when next button is clicked', () => {
    renderComponent();

    fireEvent.click(screen.getByTestId('details-pane-next-button'));

    expect(defaultProps.onNext).toHaveBeenCalledTimes(1);
  });

  it('calls onPrevious when previous button is clicked', () => {
    renderComponent();

    fireEvent.click(screen.getByTestId('details-pane-prev-button'));

    expect(defaultProps.onPrevious).toHaveBeenCalledTimes(1);
  });

  it('disables previous button when isFirstQuery is true', () => {
    render(
      <ThemeProvider theme={createTheme({ palette: { mode: 'light' } })}>
        <DetailsPane {...defaultProps} isFirstQuery />
      </ThemeProvider>
    );

    expect(screen.getByTestId('details-pane-prev-button')).toBeDisabled();
  });

  it('disables next button when isLastQuery is true', () => {
    render(
      <ThemeProvider theme={createTheme({ palette: { mode: 'light' } })}>
        <DetailsPane {...defaultProps} isLastQuery />
      </ThemeProvider>
    );

    expect(screen.getByTestId('details-pane-next-button')).toBeDisabled();
  });

  it('calls onClose when Escape key is pressed', () => {
    renderComponent();

    fireEvent.keyDown(document, { key: 'Escape' });

    expect(defaultProps.onClose).toHaveBeenCalledTimes(1);
  });

  it('shows details tab content when on first tab', () => {
    renderComponent();

    expect(
      screen.getByText((content) => content.includes('mycollection'))
    ).toBeInTheDocument();
  });

  it('shows raw data tab content when switching to raw data tab', () => {
    renderComponent();

    fireEvent.click(screen.getByTestId('details-pane-raw-data-tab'));

    expect(
      screen.getByText((content) => content.includes('mycollection'))
    ).toBeInTheDocument();
  });

  it('renders close button ', () => {
    renderComponent();

    expect(
      screen.getByRole('button', { name: Messages.actions.close })
    ).toBeInTheDocument();
  });

  it('renders prev/next buttons', () => {
    renderComponent();

    expect(
      screen.getByRole('button', { name: Messages.actions.previous })
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: Messages.actions.next })
    ).toBeInTheDocument();
  });
});

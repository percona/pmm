import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import QanHeaderActions from './QanHeaderActions';
import {
  wrapWithQueryProvider,
  wrapWithSnackbarProvider,
} from 'utils/testUtils';

const { mockCreateShortUrl } = vi.hoisted(() => ({
  mockCreateShortUrl: vi.fn().mockResolvedValue({
    uid: '1',
    url: 'https://www.percona.com/',
  }),
}));

vi.mock('api/short-urls', () => ({
  createShortUrl: mockCreateShortUrl,
}));

describe('QanHeaderActions', () => {
  beforeEach(() => {
    mockCreateShortUrl.mockClear();
  });

  it('should render', () => {
    render(wrapWithQueryProvider(<QanHeaderActions />));

    expect(
      screen.getByTestId('qan-header-actions-running-agents-button')
    ).toBeInTheDocument();
    expect(
      screen.getByTestId('qan-header-actions-copy-button')
    ).toBeInTheDocument();
  });

  it('should open running agents modal', () => {
    render(wrapWithQueryProvider(<QanHeaderActions />));

    fireEvent.click(
      screen.getByTestId('qan-header-actions-running-agents-button')
    );

    expect(screen.getByTestId('running-agents-modal')).toBeInTheDocument();
  });

  it('should close running agents modal', () => {
    render(wrapWithQueryProvider(<QanHeaderActions />));

    fireEvent.click(
      screen.getByTestId('qan-header-actions-running-agents-button')
    );

    expect(screen.getByTestId('running-agents-modal')).toBeInTheDocument();

    fireEvent.click(
      screen.getByTestId('running-agents-modal-close-window-button')
    );

    expect(
      screen.queryByTestId('running-agents-modal')
    ).not.toBeInTheDocument();
  });

  it('should copy link to clipboard (native ui)', () => {
    Object.defineProperty(window, 'location', {
      value: {
        pathname: '/pmm-ui/rta',
        search: '',
        hash: '',
      },
      writable: true,
      configurable: true,
    });

    render(
      wrapWithSnackbarProvider(wrapWithQueryProvider(<QanHeaderActions />))
    );

    fireEvent.click(screen.getByTestId('qan-header-actions-copy-button'));

    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
      window.location.href
    );

    expect(screen.getByText('Link copied to clipboard')).toBeInTheDocument();
  });

  it('should copy link to clipboard (grafana)', async () => {
    Object.defineProperty(window, 'location', {
      value: {
        pathname: '/pmm-ui/next/graph/pmm-qan/pmm-query-analytics',
        search: '',
        hash: '',
      },
    });

    render(
      wrapWithSnackbarProvider(wrapWithQueryProvider(<QanHeaderActions />))
    );

    fireEvent.click(screen.getByTestId('qan-header-actions-copy-button'));

    await waitFor(() =>
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
        'https://www.percona.com/'
      )
    );

    expect(screen.getByText('Link copied to clipboard')).toBeInTheDocument();
  });

  it('should show error message if failed to copy link to clipboard', async () => {
    mockCreateShortUrl.mockRejectedValue(
      new Error('Failed to create short url')
    );

    render(
      wrapWithSnackbarProvider(wrapWithQueryProvider(<QanHeaderActions />))
    );

    fireEvent.click(screen.getByTestId('qan-header-actions-copy-button'));

    await waitFor(() =>
      expect(
        screen.getByText('Failed to copy link to clipboard')
      ).toBeInTheDocument()
    );
  });
});

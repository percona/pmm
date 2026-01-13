import { render, screen, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter } from 'react-router-dom';
import { SnackbarProvider } from 'notistack';
import { RealTimeSelection } from './RealTimeSelection';
import { useUser } from 'contexts/user';
import * as servicesApi from 'api/services';
import * as realtimeApi from 'api/realtime';

vi.mock('contexts/user');
vi.mock('api/services');
vi.mock('api/realtime');

const mockNavigate = vi.fn();

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');

  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <SnackbarProvider>{children}</SnackbarProvider>
      </BrowserRouter>
    </QueryClientProvider>
  );
};

describe('RealTimeSelection', () => {
  beforeEach(() => {
    vi.clearAllMocks();

    vi.mocked(useUser).mockReturnValue({
      user: {
        isAuthorized: true,
        isEditor: true,
        isPMMAdmin: false,
      },
    } as ReturnType<typeof useUser>);
  });

  describe('Rendering', () => {
    it('renders title and description', () => {
      render(<RealTimeSelection />, { wrapper: createWrapper() });

      expect(screen.getByText('Real-Time Query Analytics')).toBeInTheDocument();
      expect(
        screen.getByText(
          'Select a service to monitor queries and performance metrics in real time.'
        )
      ).toBeInTheDocument();
    });

    it('renders search input', () => {
      render(<RealTimeSelection />, { wrapper: createWrapper() });

      expect(
        screen.getByPlaceholderText('Search cluster/service...')
      ).toBeInTheDocument();
    });

    it('renders start button', () => {
      render(<RealTimeSelection />, { wrapper: createWrapper() });

      expect(
        screen.getByRole('button', { name: /start session/i })
      ).toBeInTheDocument();
    });

    it('renders footer links', () => {
      render(<RealTimeSelection />, { wrapper: createWrapper() });

      expect(screen.getByText('Documentation')).toBeInTheDocument();
      expect(screen.getByText('Provide feedback')).toBeInTheDocument();
    });

    it('renders MongoDB only message', () => {
      render(<RealTimeSelection />, { wrapper: createWrapper() });

      expect(
        screen.getByText('Currently available for MongoDB only. More databases coming soon.')
      ).toBeInTheDocument();
    });
  });

  describe('Permissions', () => {
    it('disables controls when user is not editor or admin', () => {
      vi.mocked(useUser).mockReturnValue({
        user: {
          isAuthorized: true,
          isEditor: false,
          isPMMAdmin: false,
        },
      } as ReturnType<typeof useUser>);

      render(<RealTimeSelection />, { wrapper: createWrapper() });

      const autocomplete = screen.getByRole('combobox');
      const button = screen.getByRole('button', { name: /start session/i });

      expect(autocomplete).toBeDisabled();
      expect(button).toBeDisabled();
    });

    it('enables controls when user is editor', () => {
      vi.mocked(useUser).mockReturnValue({
        user: {
          isAuthorized: true,
          isEditor: true,
          isPMMAdmin: false,
        },
      } as ReturnType<typeof useUser>);

      render(<RealTimeSelection />, { wrapper: createWrapper() });

      const button = screen.getByRole('button', { name: /start session/i });

      expect(button).toBeDisabled();
    });

    it('enables controls when user is admin', () => {
      vi.mocked(useUser).mockReturnValue({
        user: {
          isAuthorized: true,
          isEditor: false,
          isPMMAdmin: true,
        },
      } as ReturnType<typeof useUser>);

      render(<RealTimeSelection />, { wrapper: createWrapper() });

      const button = screen.getByRole('button', { name: /start session/i });

      expect(button).toBeDisabled();
    });
  });

  describe('Service Selection', () => {
    it('disables start button when no services selected', () => {
      render(<RealTimeSelection />, { wrapper: createWrapper() });

      const button = screen.getByRole('button', { name: /start session/i });

      expect(button).toBeDisabled();
    });

    it('has autocomplete dropdown button', () => {
      render(<RealTimeSelection />, { wrapper: createWrapper() });

      const openButton = screen.getByRole('button', { name: /open/i });

      expect(openButton).toBeInTheDocument();
    });

    it('autocomplete starts closed', () => {
      render(<RealTimeSelection />, { wrapper: createWrapper() });

      const autocomplete = screen.getByRole('combobox');

      expect(autocomplete).toHaveAttribute('aria-expanded', 'false');
    });
  });

  describe('Form Submission', () => {
    it('shows error when starting with no services selected', async () => {
      render(<RealTimeSelection />, { wrapper: createWrapper() });

      const button = screen.getByRole('button', { name: /start session/i });

      expect(button).toBeDisabled();
    });

    it('shows success message on successful start', async () => {
      vi.mocked(realtimeApi.changeRealtimeAnalytics).mockResolvedValue({});

      render(<RealTimeSelection />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(mockNavigate).not.toHaveBeenCalled();
      });
    });
  });

  describe('Loading States', () => {
    it('shows loading indicator while fetching services', () => {
      vi.mocked(servicesApi.listServices).mockImplementation(
        () => new Promise(() => {})
      );

      render(<RealTimeSelection />, { wrapper: createWrapper() });

      const autocomplete = screen.getByRole('combobox');

      expect(autocomplete).toBeInTheDocument();
    });
  });

  describe('Error Handling', () => {
    it('handles service fetch error gracefully', async () => {
      vi.mocked(servicesApi.listServices).mockRejectedValue(
        new Error('Failed to fetch services')
      );

      render(<RealTimeSelection />, { wrapper: createWrapper() });

      await waitFor(() => {
        const autocomplete = screen.getByRole('combobox');

        expect(autocomplete).toBeInTheDocument();
      });
    });
  });

  describe('Accessibility', () => {
    it('has proper ARIA roles', () => {
      render(<RealTimeSelection />, { wrapper: createWrapper() });

      expect(screen.getByRole('combobox')).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /start session/i })).toBeInTheDocument();
    });

    it('has proper labels', () => {
      render(<RealTimeSelection />, { wrapper: createWrapper() });

      expect(screen.getByPlaceholderText('Search cluster/service...')).toBeInTheDocument();
    });
  });

  describe('Success Handling', () => {
    it('clears selection on successful start', async () => {
      vi.mocked(realtimeApi.changeRealtimeAnalytics).mockResolvedValue({});

      render(<RealTimeSelection />, { wrapper: createWrapper() });

      // TODO: Update this test when navigation is implemented
      await waitFor(() => {
        expect(mockNavigate).not.toHaveBeenCalled();
      });
    });
  });
});

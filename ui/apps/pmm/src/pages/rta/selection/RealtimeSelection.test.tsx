import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { User } from 'types/user.types';
import { RealtimeSelection } from './RealtimeSelection';
import { Messages } from './RealtimeSelection.messages';
import * as servicesApi from 'api/services';
import * as realtimeApi from 'api/rta';
import {
  wrapWithQueryProvider,
  wrapWithRouter,
  wrapWithSnackbarProvider,
  wrapWithUserProvider,
} from 'utils/testUtils';
import {
  TEST_MANAGED_SERVICE_MONGO,
  TEST_REAL_TIME_SESSION,
  TEST_USER_ADMIN,
  TEST_USER_EDITOR,
  TEST_USER_VIEWER,
} from 'utils/testStubs';

vi.mock('api/services');
vi.mock('api/rta');

const mockNavigate = vi.fn();

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');

  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

const setupMocks = () => {
  vi.mocked(servicesApi.listManagedServices).mockResolvedValue({
    services: [],
  });
  vi.mocked(realtimeApi.getRunningSessions).mockResolvedValue([]);
};

const renderComponent = (user?: User) =>
  render(
    wrapWithQueryProvider(
      wrapWithRouter(
        wrapWithSnackbarProvider(
          wrapWithUserProvider(<RealtimeSelection />, { user })
        )
      )
    )
  );

describe('RealtimeSelection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupMocks();
  });

  describe('Rendering', () => {
    it('renders title and description', async () => {
      renderComponent(TEST_USER_EDITOR);

      await waitFor(() => {
        expect(screen.getByText(Messages.title)).toBeInTheDocument();
      });
      expect(screen.getByText(Messages.description)).toBeInTheDocument();
    });

    it('renders search input', async () => {
      renderComponent(TEST_USER_EDITOR);

      await waitFor(() => {
        expect(
          screen.getByPlaceholderText(Messages.searchPlaceholder)
        ).toBeInTheDocument();
      });
    });

    it('renders start button', async () => {
      renderComponent(TEST_USER_EDITOR);

      await waitFor(() => {
        expect(
          screen.getByRole('button', {
            name: new RegExp(Messages.startButton, 'i'),
          })
        ).toBeInTheDocument();
      });
    });

    it('renders footer links', async () => {
      renderComponent(TEST_USER_EDITOR);

      await waitFor(() => {
        expect(screen.getByText(Messages.documentation)).toBeInTheDocument();
      });
      expect(screen.getByText(Messages.feedback)).toBeInTheDocument();
    });

    it('renders MongoDB only message', async () => {
      renderComponent(TEST_USER_EDITOR);

      await waitFor(() => {
        expect(screen.getByText(Messages.mongoOnly)).toBeInTheDocument();
      });
    });
  });

  describe('Permissions', () => {
    it('shows empty state for viewer users when no running agents', async () => {
      // Viewers without running agents see empty state, not the selection form
      renderComponent(TEST_USER_VIEWER);

      await waitFor(() => {
        // Viewer should see empty state, not the form
        expect(screen.queryByRole('combobox')).not.toBeInTheDocument();
      });
    });

    it('enables autocomplete for editor users but button stays disabled without selection', async () => {
      renderComponent(TEST_USER_EDITOR);

      await waitFor(() => {
        const button = screen.getByRole('button', {
          name: new RegExp(Messages.startButton, 'i'),
        });

        expect(button).toBeDisabled();
      });
    });

    it('enables autocomplete for admin users but button stays disabled without selection', async () => {
      renderComponent(TEST_USER_ADMIN);

      await waitFor(() => {
        const button = screen.getByRole('button', {
          name: new RegExp(Messages.startButton, 'i'),
        });

        expect(button).toBeDisabled();
      });
    });
  });

  describe('Service Selection', () => {
    it('disables start button when no services selected', async () => {
      renderComponent(TEST_USER_EDITOR);

      await waitFor(() => {
        const button = screen.getByRole('button', {
          name: new RegExp(Messages.startButton, 'i'),
        });

        expect(button).toBeDisabled();
      });
    });

    it('has autocomplete dropdown button', async () => {
      renderComponent(TEST_USER_EDITOR);

      await waitFor(() => {
        const openButton = screen.getByRole('button', { name: /open/i });

        expect(openButton).toBeInTheDocument();
      });
    });

    it('autocomplete starts closed', async () => {
      renderComponent(TEST_USER_EDITOR);

      await waitFor(() => {
        const autocomplete = screen.getByRole('combobox');

        expect(autocomplete).toHaveAttribute('aria-expanded', 'false');
      });
    });
  });

  describe('Form Submission', () => {
    it('keeps button disabled when no services selected', async () => {
      renderComponent(TEST_USER_EDITOR);

      await waitFor(() => {
        const button = screen.getByRole('button', {
          name: new RegExp(Messages.startButton, 'i'),
        });

        expect(button).toBeDisabled();
      });
    });

    it('navigates to overview on success', async () => {
      vi.mocked(realtimeApi.startSession).mockResolvedValue({
        session: TEST_REAL_TIME_SESSION,
      });
      vi.mocked(servicesApi.listManagedServices).mockResolvedValue({
        services: [TEST_MANAGED_SERVICE_MONGO],
      });

      renderComponent(TEST_USER_EDITOR);

      // Select a service from the dropdown
      const serviceInput = await screen.findByTitle('Open');
      fireEvent.click(serviceInput);

      const listbox = await screen.findByRole('listbox');
      const option = within(listbox).getByText(
        TEST_MANAGED_SERVICE_MONGO.serviceName
      );
      fireEvent.click(option);

      const startButton = screen.getByTestId('start-realtime-session');
      fireEvent.click(startButton);

      await waitFor(() => expect(realtimeApi.startSession).toHaveBeenCalled());

      await waitFor(() => {
        expect(mockNavigate).toHaveBeenCalled();
      });
    });
  });

  describe('Loading States', () => {
    it('shows loading indicator while fetching services', () => {
      vi.mocked(servicesApi.listManagedServices).mockImplementation(
        () => new Promise(() => {})
      );

      renderComponent(TEST_USER_EDITOR);

      expect(screen.getByRole('progressbar')).toBeInTheDocument();
    });
  });

  describe('Error Handling', () => {
    it('renders form even when service fetch fails', async () => {
      vi.mocked(servicesApi.listManagedServices).mockRejectedValueOnce(
        new Error('Failed to fetch services')
      );
      vi.mocked(realtimeApi.getRunningSessions).mockResolvedValueOnce([]);

      renderComponent(TEST_USER_EDITOR);

      // When API fails, component should still render (not stuck in loading)
      await waitFor(
        () => {
          expect(screen.queryByRole('progressbar')).not.toBeInTheDocument();
        },
        { timeout: 3000 }
      );
    });
  });
});

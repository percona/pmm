import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { RealtimeSelection } from './RealtimeSelection';
import { Messages } from './RealtimeSelection.messages';
import * as realtimeApi from 'api/rta';
import {
  wrapWithQueryProvider,
  wrapWithSnackbarProvider,
  wrapWithUserProvider,
} from 'utils/testUtils';
import {
  TEST_VERSIONED_MONGO_SERVICE,
  TEST_REAL_TIME_SESSION,
  TEST_USER_ADMIN,
  TEST_USER_EDITOR,
  TEST_USER_VIEWER,
} from 'utils/testStubs';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { Messages as RtaMessages } from '../messages';

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
  vi.mocked(realtimeApi.getRunningSessions).mockResolvedValue([]);
  vi.mocked(realtimeApi.getAvailableServices).mockResolvedValue({
    mongodb: [],
  });
};

const renderComponent = (user = TEST_USER_ADMIN) =>
  render(
    wrapWithQueryProvider(
      wrapWithSnackbarProvider(
        wrapWithUserProvider(
          <MemoryRouter initialEntries={['/rta/selection']} initialIndex={0}>
            <Routes>
              <Route path="/rta/selection" element={<RealtimeSelection />} />
              <Route
                path="/rta/sessions"
                element={<div data-testid="realtime-sessions">Sessions</div>}
              />
            </Routes>
          </MemoryRouter>,
          { user }
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
      renderComponent();

      await waitFor(() => {
        expect(screen.getByText(Messages.title)).toBeInTheDocument();
      });
      expect(screen.getByText(Messages.description)).toBeInTheDocument();
    });

    it('renders search input', async () => {
      renderComponent();

      await waitFor(() => {
        expect(
          screen.getByPlaceholderText(Messages.searchPlaceholder)
        ).toBeInTheDocument();
      });
    });

    it('renders start button', async () => {
      renderComponent();

      await waitFor(() => {
        expect(
          screen.getByRole('button', {
            name: new RegExp(Messages.startButton, 'i'),
          })
        ).toBeInTheDocument();
      });
    });

    it('renders footer links', async () => {
      renderComponent();

      await waitFor(() => {
        expect(screen.getByText(Messages.documentation)).toBeInTheDocument();
      });
      expect(screen.getByText(Messages.feedback)).toBeInTheDocument();
    });

    it('renders disclaimer message', async () => {
      renderComponent();

      await waitFor(() => {
        expect(screen.getByText(RtaMessages.disclaimer)).toBeInTheDocument();
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

    it('shows empty state for editor users when no running agents', async () => {
      // Editors without running agents see empty state, not the selection form
      renderComponent(TEST_USER_EDITOR);

      await waitFor(() => {
        // Editor should see empty state, not the form
        expect(screen.queryByRole('combobox')).not.toBeInTheDocument();
      });
    });

    it('enables autocomplete for admin users but button stays disabled without selection', async () => {
      renderComponent();

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
      renderComponent();

      await waitFor(() => {
        const button = screen.getByRole('button', {
          name: new RegExp(Messages.startButton, 'i'),
        });

        expect(button).toBeDisabled();
      });
    });

    it('has autocomplete dropdown button', async () => {
      renderComponent();

      await waitFor(() => {
        const openButton = screen.getByRole('button', { name: /open/i });

        expect(openButton).toBeInTheDocument();
      });
    });

    it('autocomplete starts closed', async () => {
      renderComponent();

      await waitFor(() => {
        const autocomplete = screen.getByRole('combobox');

        expect(autocomplete).toHaveAttribute('aria-expanded', 'false');
      });
    });
  });

  describe('Form Submission', () => {
    it('keeps button disabled when no services selected', async () => {
      renderComponent();

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
      vi.mocked(realtimeApi.getAvailableServices).mockResolvedValue({
        mongodb: [TEST_VERSIONED_MONGO_SERVICE],
      });

      renderComponent();

      // Select a service from the dropdown
      const serviceInput = await screen.findByTitle('Open');
      fireEvent.click(serviceInput);

      const listbox = await screen.findByRole('listbox');
      const option = within(listbox).getByText(
        TEST_VERSIONED_MONGO_SERVICE.serviceName
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
      vi.mocked(realtimeApi.getAvailableServices).mockImplementation(
        () => new Promise(() => {})
      );

      renderComponent();

      expect(screen.getByRole('progressbar')).toBeInTheDocument();
    });
  });

  describe('Error Handling', () => {
    it('renders form even when service fetch fails', async () => {
      vi.mocked(realtimeApi.getAvailableServices).mockRejectedValueOnce(
        new Error('Failed to fetch services')
      );
      vi.mocked(realtimeApi.getRunningSessions).mockResolvedValueOnce([]);

      renderComponent();

      // When API fails, component should still render (not stuck in loading)
      await waitFor(
        () => {
          expect(screen.queryByRole('progressbar')).not.toBeInTheDocument();
        },
        { timeout: 3000 }
      );
    });
  });

  describe('Navigation', () => {
    it('navigates to sessions page when there are any running sessions', async () => {
      vi.mocked(realtimeApi.getRunningSessions).mockResolvedValueOnce([
        TEST_REAL_TIME_SESSION,
      ]);

      renderComponent();

      await waitFor(() =>
        expect(screen.getByTestId('realtime-sessions')).toBeInTheDocument()
      );
    });
  });
});

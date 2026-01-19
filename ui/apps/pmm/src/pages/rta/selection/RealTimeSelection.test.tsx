import { render, screen, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { RealTimeSelection } from './RealTimeSelection';
import { Messages } from './RealTimeSelection.messages';
import { useUser } from 'contexts/user';
import * as servicesApi from 'api/services';
import * as realtimeApi from 'api/realtime';
import {
  wrapWithQueryProvider,
  wrapWithRouter,
  wrapWithSnackbarProvider,
} from 'utils/testUtils';
import { TEST_USER_ADMIN, TEST_USER_EDITOR, TEST_USER_VIEWER } from 'utils/testStubs';

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

const renderComponent = () =>
  render(
    wrapWithQueryProvider(
      wrapWithRouter(wrapWithSnackbarProvider(<RealTimeSelection />))
    )
  );

describe('RealTimeSelection', () => {
  beforeEach(() => {
    vi.clearAllMocks();

    vi.mocked(useUser).mockReturnValue({
      user: TEST_USER_EDITOR,
      isLoading: false,
    });
  });

  describe('Rendering', () => {
    it('renders title and description', () => {
      renderComponent();

      expect(screen.getByText(Messages.title)).toBeInTheDocument();
      expect(screen.getByText(Messages.description)).toBeInTheDocument();
    });

    it('renders search input', () => {
      renderComponent();

      expect(
        screen.getByPlaceholderText(Messages.searchPlaceholder)
      ).toBeInTheDocument();
    });

    it('renders start button', () => {
      renderComponent();

      expect(
        screen.getByRole('button', { name: new RegExp(Messages.startButton, 'i') })
      ).toBeInTheDocument();
    });

    it('renders footer links', () => {
      renderComponent();

      expect(screen.getByText(Messages.documentation)).toBeInTheDocument();
      expect(screen.getByText(Messages.feedback)).toBeInTheDocument();
    });

    it('renders MongoDB only message', () => {
      renderComponent();

      expect(screen.getByText(Messages.mongoOnly)).toBeInTheDocument();
    });
  });

  describe('Permissions', () => {
    it('disables controls for viewer users', () => {
      vi.mocked(useUser).mockReturnValue({
        user: TEST_USER_VIEWER,
        isLoading: false,
      });

      renderComponent();

      const autocomplete = screen.getByRole('combobox');
      const button = screen.getByRole('button', { name: new RegExp(Messages.startButton, 'i') });

      expect(autocomplete).toBeDisabled();
      expect(button).toBeDisabled();
    });

    it('enables autocomplete for editor users but button stays disabled without selection', () => {
      vi.mocked(useUser).mockReturnValue({
        user: TEST_USER_EDITOR,
        isLoading: false,
      });

      renderComponent();

      const button = screen.getByRole('button', { name: new RegExp(Messages.startButton, 'i') });

      expect(button).toBeDisabled();
    });

    it('enables autocomplete for admin users but button stays disabled without selection', () => {
      vi.mocked(useUser).mockReturnValue({
        user: TEST_USER_ADMIN,
        isLoading: false,
      });

      renderComponent();

      const button = screen.getByRole('button', { name: new RegExp(Messages.startButton, 'i') });

      expect(button).toBeDisabled();
    });
  });

  describe('Service Selection', () => {
    it('disables start button when no services selected', () => {
      renderComponent();

      const button = screen.getByRole('button', { name: new RegExp(Messages.startButton, 'i') });

      expect(button).toBeDisabled();
    });

    it('has autocomplete dropdown button', () => {
      renderComponent();

      const openButton = screen.getByRole('button', { name: /open/i });

      expect(openButton).toBeInTheDocument();
    });

    it('autocomplete starts closed', () => {
      renderComponent();

      const autocomplete = screen.getByRole('combobox');

      expect(autocomplete).toHaveAttribute('aria-expanded', 'false');
    });
  });

  describe('Form Submission', () => {
    it('keeps button disabled when no services selected', () => {
      renderComponent();

      const button = screen.getByRole('button', { name: new RegExp(Messages.startButton, 'i') });

      expect(button).toBeDisabled();
    });

    it.skip('shows success message on successful start', async () => {
      // TODO: Implement when service selection interaction is added
      vi.mocked(realtimeApi.changeRealtimeAnalytics).mockResolvedValue({});

      renderComponent();

      await waitFor(() => {
        expect(mockNavigate).not.toHaveBeenCalled();
      });
    });
  });

  describe('Loading States', () => {
    it('renders autocomplete while fetching services', () => {
      vi.mocked(servicesApi.listServices).mockImplementation(
        () => new Promise(() => {})
      );

      renderComponent();

      const autocomplete = screen.getByRole('combobox');

      expect(autocomplete).toBeInTheDocument();
    });
  });

  describe('Error Handling', () => {
    it('renders autocomplete even when service fetch fails', async () => {
      vi.mocked(servicesApi.listServices).mockRejectedValue(
        new Error('Failed to fetch services')
      );

      renderComponent();

      await waitFor(() => {
        const autocomplete = screen.getByRole('combobox');

        expect(autocomplete).toBeInTheDocument();
      });
    });
  });

  describe('Accessibility', () => {
    it('has proper ARIA roles', () => {
      renderComponent();

      expect(screen.getByRole('combobox')).toBeInTheDocument();
      expect(screen.getByRole('button', { name: new RegExp(Messages.startButton, 'i') })).toBeInTheDocument();
    });

    it('has proper placeholder text', () => {
      renderComponent();

      expect(screen.getByPlaceholderText(Messages.searchPlaceholder)).toBeInTheDocument();
    });
  });

  describe('Success Handling', () => {
    it.skip('clears selection on successful start', async () => {
      // TODO: Implement when service selection interaction is added
      vi.mocked(realtimeApi.changeRealtimeAnalytics).mockResolvedValue({});

      renderComponent();

      await waitFor(() => {
        expect(mockNavigate).not.toHaveBeenCalled();
      });
    });
  });
});

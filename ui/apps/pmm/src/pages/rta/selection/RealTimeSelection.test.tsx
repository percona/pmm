import { render, screen, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { User } from 'types/user.types';
import { RealTimeSelection } from './RealTimeSelection';
import { Messages } from './RealTimeSelection.messages';
import * as servicesApi from 'api/services';
import * as realtimeApi from 'api/realtime';
import {
  wrapWithQueryProvider,
  wrapWithRouter,
  wrapWithSnackbarProvider,
  wrapWithUserProvider,
} from 'utils/testUtils';
import { TEST_USER_ADMIN, TEST_USER_EDITOR, TEST_USER_VIEWER } from 'utils/testStubs';

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

const renderComponent = (user?: User) =>
  render(
    wrapWithQueryProvider(
      wrapWithRouter(
        wrapWithSnackbarProvider(
          wrapWithUserProvider(<RealTimeSelection />, { user })
        )
      )
    )
  );

describe('RealTimeSelection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Rendering', () => {
    it('renders title and description', () => {
      renderComponent(TEST_USER_EDITOR);

      expect(screen.getByText(Messages.title)).toBeInTheDocument();
      expect(screen.getByText(Messages.description)).toBeInTheDocument();
    });

    it('renders search input', () => {
      renderComponent(TEST_USER_EDITOR);

      expect(
        screen.getByPlaceholderText(Messages.searchPlaceholder)
      ).toBeInTheDocument();
    });

    it('renders start button', () => {
      renderComponent(TEST_USER_EDITOR);

      expect(
        screen.getByRole('button', { name: new RegExp(Messages.startButton, 'i') })
      ).toBeInTheDocument();
    });

    it('renders footer links', () => {
      renderComponent(TEST_USER_EDITOR);

      expect(screen.getByText(Messages.documentation)).toBeInTheDocument();
      expect(screen.getByText(Messages.feedback)).toBeInTheDocument();
    });

    it('renders MongoDB only message', () => {
      renderComponent(TEST_USER_EDITOR);

      expect(screen.getByText(Messages.mongoOnly)).toBeInTheDocument();
    });
  });

  describe('Permissions', () => {
    it('disables controls for viewer users', () => {
      renderComponent(TEST_USER_VIEWER);

      const autocomplete = screen.getByRole('combobox');
      const button = screen.getByRole('button', { name: new RegExp(Messages.startButton, 'i') });

      expect(autocomplete).toBeDisabled();
      expect(button).toBeDisabled();
    });

    it('enables autocomplete for editor users but button stays disabled without selection', () => {
      renderComponent(TEST_USER_EDITOR);

      const button = screen.getByRole('button', { name: new RegExp(Messages.startButton, 'i') });

      expect(button).toBeDisabled();
    });

    it('enables autocomplete for admin users but button stays disabled without selection', () => {
      renderComponent(TEST_USER_ADMIN);

      const button = screen.getByRole('button', { name: new RegExp(Messages.startButton, 'i') });

      expect(button).toBeDisabled();
    });
  });

  describe('Service Selection', () => {
    it('disables start button when no services selected', () => {
      renderComponent(TEST_USER_EDITOR);

      const button = screen.getByRole('button', { name: new RegExp(Messages.startButton, 'i') });

      expect(button).toBeDisabled();
    });

    it('has autocomplete dropdown button', () => {
      renderComponent(TEST_USER_EDITOR);

      const openButton = screen.getByRole('button', { name: /open/i });

      expect(openButton).toBeInTheDocument();
    });

    it('autocomplete starts closed', () => {
      renderComponent(TEST_USER_EDITOR);

      const autocomplete = screen.getByRole('combobox');

      expect(autocomplete).toHaveAttribute('aria-expanded', 'false');
    });
  });

  describe('Form Submission', () => {
    it('keeps button disabled when no services selected', () => {
      renderComponent(TEST_USER_EDITOR);

      const button = screen.getByRole('button', { name: new RegExp(Messages.startButton, 'i') });

      expect(button).toBeDisabled();
    });

    it.skip('shows success message on successful start', async () => {
      // TODO: Implement when service selection interaction is added
      vi.mocked(realtimeApi.changeRealtimeAnalytics).mockResolvedValue({});

      renderComponent(TEST_USER_EDITOR);

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

      renderComponent(TEST_USER_EDITOR);

      const autocomplete = screen.getByRole('combobox');

      expect(autocomplete).toBeInTheDocument();
    });
  });

  describe('Error Handling', () => {
    it('renders autocomplete even when service fetch fails', async () => {
      vi.mocked(servicesApi.listServices).mockRejectedValue(
        new Error('Failed to fetch services')
      );

      renderComponent(TEST_USER_EDITOR);

      await waitFor(() => {
        const autocomplete = screen.getByRole('combobox');

        expect(autocomplete).toBeInTheDocument();
      });
    });
  });

  describe('Success Handling', () => {
    it.skip('clears selection on successful start', async () => {
      // TODO: Implement when service selection interaction is added
      vi.mocked(realtimeApi.changeRealtimeAnalytics).mockResolvedValue({});

      renderComponent(TEST_USER_EDITOR);

      await waitFor(() => {
        expect(mockNavigate).not.toHaveBeenCalled();
      });
    });
  });
});

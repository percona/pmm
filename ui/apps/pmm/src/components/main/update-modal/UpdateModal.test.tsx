import { render, screen, fireEvent } from '@testing-library/react';
import { vi } from 'vitest';
import UpdateModal from './UpdateModal';
import { TestWrapper } from 'utils/testWrapper';
import {
  wrapWithUpdatesProvider,
  wrapWithQueryProvider,
  wrapWithSettings,
} from 'utils/testUtils';

// Mock the snooze hook
const mockSnoozeUpdate = vi.fn();
const mocks = vi.hoisted(() => ({
  mockUseSnooze: vi.fn(),
}));

vi.mock('hooks/updates', () => ({
  useSnooze: mocks.mockUseSnooze,
}));

const mockVersionInfo = {
  installed: {
    version: '3.0.0',
    fullVersion: '3.0.0',
    timestamp: new Date().toISOString(),
  },
  latest: {
    version: '3.1.0',
    tag: '',
    timestamp: null,
    releaseNotesText: '',
    releaseNotesUrl: 'https://example.com/release-notes',
  },
  updateAvailable: true,
  latestNewsUrl: 'https://per.co.na/pmm/3.1.0',
  lastCheck: '2024-07-30T10:34:05.886739003Z',
};

const renderUpdateModal = (
  overrides = {},
  snoozeCount = 0,
  initialEntries = ['/']
) => {
  const defaultProps = {
    isLoading: false,
    versionInfo: mockVersionInfo,
    ...overrides,
  };

  // Update the mock to return the specified snooze count
  mocks.mockUseSnooze.mockReturnValue({
    snoozeUpdate: mockSnoozeUpdate,
    snoozedAt: '',
    snoozeActive: false,
    snoozeCount,
  });

  return render(
    <TestWrapper
      routerProps={{
        initialEntries,
      }}
    >
      {wrapWithQueryProvider(
        wrapWithSettings(wrapWithUpdatesProvider(<UpdateModal />, defaultProps))
      )}
    </TestWrapper>
  );
};

describe('UpdateModal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Component visibility conditions', () => {
    it('renders nothing when loading', () => {
      renderUpdateModal({ isLoading: true });
      expect(screen.queryByTestId('modal-title')).not.toBeInTheDocument();
    });

    it('renders nothing when no version info', () => {
      renderUpdateModal({ versionInfo: undefined });
      expect(screen.queryByTestId('modal-title')).not.toBeInTheDocument();
    });

    it('renders nothing when no update available', () => {
      renderUpdateModal({
        versionInfo: { ...mockVersionInfo, updateAvailable: false },
      });
      expect(screen.queryByTestId('modal-title')).not.toBeInTheDocument();
    });

    it('renders nothing when already on updates page', () => {
      renderUpdateModal({}, 0, ['/updates']);
      expect(screen.queryByTestId('modal-title')).not.toBeInTheDocument();
    });
  });

  describe('Component structure', () => {
    it('renders modal for first-time users (snoozeCount = 0)', () => {
      renderUpdateModal({}, 0);

      expect(screen.getByTestId('modal-title')).toBeInTheDocument();
      expect(
        screen.getByTestId('update-modal-description')
      ).toBeInTheDocument();
      expect(
        screen.getByTestId('update-modal-description-release-notes')
      ).toBeInTheDocument();
      expect(
        screen.getByTestId('update-modal-release-notes-link')
      ).toBeInTheDocument();
      expect(
        screen.getByTestId('update-modal-remind-me-button')
      ).toBeInTheDocument();
      expect(
        screen.getByTestId('update-modal-go-to-updates-button')
      ).toBeInTheDocument();
    });

    it('renders snackbar for users with snoozeCount > 1', () => {
      renderUpdateModal({}, 2);

      expect(screen.getByTestId('update-modal-snackbar')).toBeInTheDocument();
      expect(screen.getByTestId('update-modal-title')).toBeInTheDocument();
      expect(
        screen.getByTestId('update-modal-snackbar-description')
      ).toBeInTheDocument();
      expect(
        screen.getByTestId('update-modal-remind-me-button')
      ).toBeInTheDocument();
      expect(
        screen.getByTestId('update-modal-go-to-updates-button')
      ).toBeInTheDocument();
      expect(
        screen.getByTestId('update-modal-close-button')
      ).toBeInTheDocument();

      // Modal should not be rendered when snoozeCount > 1
      expect(screen.queryByTestId('modal-title')).not.toBeInTheDocument();
      expect(
        screen.queryByTestId('update-modal-description')
      ).not.toBeInTheDocument();
    });

    it('has correct release notes link in modal', () => {
      renderUpdateModal({}, 0);

      const releaseNotesLink = screen.getByTestId(
        'update-modal-release-notes-link'
      );
      expect(releaseNotesLink).toHaveAttribute(
        'href',
        'https://example.com/release-notes'
      );
      expect(releaseNotesLink).toHaveAttribute('target', '_blank');
      expect(releaseNotesLink).toHaveAttribute('rel', 'noopener noreferrer');
    });

    it('renders modal with correct title when snoozeCount = 0', () => {
      renderUpdateModal({}, 0);

      const titleElement = screen.getByTestId('modal-title');
      expect(titleElement).toBeInTheDocument();
      expect(titleElement).toHaveTextContent('Update to PMM 3.1.0');
    });

    it('renders snackbar with correct title when snoozeCount > 1', () => {
      renderUpdateModal({}, 2);

      const titleElement = screen.getByTestId('update-modal-title');
      expect(titleElement).toBeInTheDocument();
      expect(titleElement).toHaveTextContent('Update to PMM 3.1.0');
    });
  });

  describe('Edge cases', () => {
    it('handles missing release notes URL gracefully in modal', () => {
      const versionInfoWithoutReleaseNotes = {
        ...mockVersionInfo,
        latest: {
          ...mockVersionInfo.latest,
          releaseNotesUrl: '',
        },
      };

      renderUpdateModal({ versionInfo: versionInfoWithoutReleaseNotes }, 0);

      expect(
        screen.getByTestId('update-modal-release-notes-link')
      ).toBeInTheDocument();
    });

    it('handles version info with null latest version in modal', () => {
      const versionInfoWithNullVersion = {
        ...mockVersionInfo,
        latest: {
          ...mockVersionInfo.latest,
          version: null,
        },
      };

      renderUpdateModal({ versionInfo: versionInfoWithNullVersion }, 0);

      const titleElement = screen.getByTestId('modal-title');
      expect(titleElement).toBeInTheDocument();
      expect(titleElement).toHaveTextContent('Update to PMM null');
    });

    it('handles version info with null latest version in snackbar', () => {
      const versionInfoWithNullVersion = {
        ...mockVersionInfo,
        latest: {
          ...mockVersionInfo.latest,
          version: null,
        },
      };

      renderUpdateModal({ versionInfo: versionInfoWithNullVersion }, 2);

      const titleElement = screen.getByTestId('update-modal-title');
      expect(titleElement).toBeInTheDocument();
      expect(titleElement).toHaveTextContent('Update to PMM null');
    });
  });

  describe('Component behavior', () => {
    it('renders modal open by default when snoozeCount = 0', () => {
      renderUpdateModal({}, 0);

      expect(screen.getByTestId('modal-title')).toBeInTheDocument();
    });

    it('renders snackbar open by default when snoozeCount > 1', () => {
      renderUpdateModal({}, 2);

      expect(screen.getByTestId('update-modal-snackbar')).toBeInTheDocument();
    });

    it('renders modal close button when snoozeCount = 0', () => {
      renderUpdateModal({}, 0);

      expect(screen.getByTestId('modal-close-button')).toBeInTheDocument();
    });

    it('renders snackbar close button when snoozeCount > 1', () => {
      renderUpdateModal({}, 2);

      expect(
        screen.getByTestId('update-modal-close-button')
      ).toBeInTheDocument();
    });

    it('snoozes update when remind me button is clicked in modal', () => {
      renderUpdateModal({}, 0);

      expect(
        screen.getByTestId('update-modal-remind-me-button')
      ).toBeInTheDocument();

      fireEvent.click(screen.getByTestId('update-modal-remind-me-button'));

      expect(mockSnoozeUpdate).toHaveBeenCalled();
    });

    it('snoozes update when remind me button is clicked in snackbar', () => {
      renderUpdateModal({}, 2);

      expect(
        screen.getByTestId('update-modal-remind-me-button')
      ).toBeInTheDocument();

      fireEvent.click(screen.getByTestId('update-modal-remind-me-button'));

      expect(mockSnoozeUpdate).toHaveBeenCalled();
    });

    it('snoozes update when modal is closed', () => {
      renderUpdateModal({}, 0);

      expect(screen.getByTestId('modal-close-button')).toBeInTheDocument();

      fireEvent.click(screen.getByTestId('modal-close-button'));

      expect(mockSnoozeUpdate).toHaveBeenCalled();
    });

    it('snoozes update when snackbar is closed', () => {
      renderUpdateModal({}, 2);

      expect(
        screen.getByTestId('update-modal-close-button')
      ).toBeInTheDocument();

      fireEvent.click(screen.getByTestId('update-modal-close-button'));

      expect(mockSnoozeUpdate).toHaveBeenCalled();
    });

    it('snoozes update when go to updates button is clicked in modal', async () => {
      renderUpdateModal({}, 0);

      expect(
        screen.getByTestId('update-modal-go-to-updates-button')
      ).toBeInTheDocument();

      fireEvent.click(screen.getByTestId('update-modal-go-to-updates-button'));

      expect(mockSnoozeUpdate).toHaveBeenCalled();
    });

    it('snoozes update when go to updates button is clicked in snackbar', async () => {
      renderUpdateModal({}, 2);

      expect(
        screen.getByTestId('update-modal-go-to-updates-button')
      ).toBeInTheDocument();

      fireEvent.click(screen.getByTestId('update-modal-go-to-updates-button'));

      expect(mockSnoozeUpdate).toHaveBeenCalled();
    });
  });
});

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
vi.mock('hooks/snooze', () => ({
  useSnooze: () => ({
    snoozeUpdate: mockSnoozeUpdate,
    snoozeActive: false,
    snoozeCount: 0,
  }),
}));

const mockVersionInfo = {
  installed: {
    version: '3.0.0',
    fullVersion: '3.0.0',
    timestamp: '2024-07-23T00:00:00Z',
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

const renderUpdateModal = (overrides = {}) => {
  const defaultProps = {
    isLoading: false,
    versionInfo: mockVersionInfo,
    ...overrides,
  };
  return render(
    <TestWrapper>
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
  });

  describe('Component structure', () => {
    it('renders modal for first-time users', () => {
      renderUpdateModal();

      expect(screen.getByTestId('modal-title')).toBeInTheDocument();
      expect(
        screen.getByTestId('update-modal-description')
      ).toBeInTheDocument();
      expect(
        screen.getByTestId('update-modal-highlights-title')
      ).toBeInTheDocument();
      expect(
        screen.getByTestId('update-modal-highlights-generic')
      ).toBeInTheDocument();
      expect(screen.getByTestId('update-modal-more-text')).toBeInTheDocument();
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

    it('has correct release notes link', () => {
      renderUpdateModal();

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

    it('renders modal with correct title', () => {
      renderUpdateModal();

      const titleElement = screen.getByTestId('modal-title');
      expect(titleElement).toBeInTheDocument();
      expect(titleElement).toHaveTextContent('Update to PMM 3.1.0');
    });
  });

  describe('Edge cases', () => {
    it('handles missing release notes URL gracefully', () => {
      const versionInfoWithoutReleaseNotes = {
        ...mockVersionInfo,
        latest: {
          ...mockVersionInfo.latest,
          releaseNotesUrl: '',
        },
      };

      renderUpdateModal({ versionInfo: versionInfoWithoutReleaseNotes });

      expect(
        screen.getByTestId('update-modal-release-notes-link')
      ).toBeInTheDocument();
    });

    it('handles version info with null latest version', () => {
      const versionInfoWithNullVersion = {
        ...mockVersionInfo,
        latest: {
          ...mockVersionInfo.latest,
          version: null as any,
        },
      };

      renderUpdateModal({ versionInfo: versionInfoWithNullVersion });

      const titleElement = screen.getByTestId('modal-title');
      expect(titleElement).toBeInTheDocument();
      expect(titleElement).toHaveTextContent('Update to PMM null');
    });
  });

  describe('Component behavior', () => {
    it('renders open by default', () => {
      renderUpdateModal();

      expect(screen.getByTestId('modal-title')).toBeInTheDocument();
    });

    it('renders modal close button', () => {
      renderUpdateModal();

      expect(screen.getByTestId('modal-close-button')).toBeInTheDocument();
    });

    it('snoozes update when remind be button is clicked', () => {
      renderUpdateModal();

      expect(
        screen.getByTestId('update-modal-remind-me-button')
      ).toBeInTheDocument();

      fireEvent.click(screen.getByTestId('update-modal-remind-me-button'));

      expect(mockSnoozeUpdate).toHaveBeenCalled();
    });

    it('snoozes update when modal is closed', () => {
      renderUpdateModal();

      expect(screen.getByTestId('modal-close-button')).toBeInTheDocument();

      fireEvent.click(screen.getByTestId('modal-close-button'));

      expect(mockSnoozeUpdate).toHaveBeenCalled();
    });

    it('snooze update when go to updates button is clicked', async () => {
      renderUpdateModal();

      expect(
        screen.getByTestId('update-modal-go-to-updates-button')
      ).toBeInTheDocument();

      fireEvent.click(screen.getByTestId('update-modal-go-to-updates-button'));

      expect(mockSnoozeUpdate).toHaveBeenCalled();
    });
  });
});

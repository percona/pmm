import { screen, render } from '@testing-library/react';
import { Footer } from './Footer';
import { Messages } from './Footer.messages';
import { wrapWithUpdatesProvider } from 'utils/testUtils';

describe('Footer', () => {
  it("doesnt't show when version info is not available", () => {
    render(
      wrapWithUpdatesProvider(<Footer />, {
        versionInfo: undefined,
      })
    );

    expect(screen.queryByTestId('pmm-footer')).toBeNull();
  });

  it('shows  correct checked date', () => {
    render(wrapWithUpdatesProvider(<Footer />));

    expect('Last checked: 2024/07/30');
  });

  it('hides the check date when no check has run', () => {
    render(
      wrapWithUpdatesProvider(<Footer />, {
        versionInfo: {
          lastCheck: null,
          latest: null,
          installed: {
            version: '3.10.0',
            fullVersion: '3.10.0',
            timestamp: '2026-07-30T00:00:00Z',
          },
          latestNewsUrl: '',
          updateAvailable: false,
        },
      })
    );

    expect(screen.getByText(Messages.version('3.10.0'))).toBeDefined();
    expect(screen.queryByText(/Last checked/)).toBeNull();
  });

  it('shows in progress message', () => {
    render(
      wrapWithUpdatesProvider(<Footer />, {
        inProgress: true,
      })
    );

    expect(screen.getByText(Messages.inProgress));
  });
});

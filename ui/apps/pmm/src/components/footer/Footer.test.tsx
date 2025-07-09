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

    expect('Checked on: 2024/07/30');
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

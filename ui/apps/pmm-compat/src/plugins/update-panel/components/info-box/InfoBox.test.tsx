import React from 'react';
import { render, screen } from '@testing-library/react';
import { InfoBox } from './InfoBox';

jest.mock('@grafana/ui', () => ({
  ...jest.requireActual('@grafana/ui'),
  Link: ({ href, children }: { href: string; children: React.ReactNode }) => (
    <a href={href}>{children}</a>
  ),
}));
import { Messages } from './InfoBox.messages';

describe('InfoBox::', () => {
  it('should show that there are no updates by default', () => {
    render(<InfoBox />);

    expect(screen.getByText(Messages.noUpdates)).toBeInTheDocument();
    expect(screen.getByText(Messages.updatesNotice)).toBeInTheDocument();
  });

  it('should show a different message if upToDate is true', () => {
    render(<InfoBox upToDate />);

    expect(screen.getByText(Messages.upToDate)).toBeInTheDocument();
  });

  it('should show an insufficient access message', () => {
    render(<InfoBox hasNoAccess />);

    expect(screen.getByText(Messages.noAccess)).toBeInTheDocument();
  });

  it('should show updates disabled messages', () => {
    render(<InfoBox updatesDisabled />);
    expect(screen.getByTestId('updates-disabled').textContent).toBe(
      `${Messages.updatesDisabled}${Messages.pmmSettings}`
    );
  });

  it('should show not online messages', () => {
    render(<InfoBox isOnline={false} />);

    expect(screen.getByText(Messages.notOnline)).toBeInTheDocument();
  });
});

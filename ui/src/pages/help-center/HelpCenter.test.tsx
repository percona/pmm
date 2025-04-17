import { render, screen } from '@testing-library/react';
import { HelpCenter } from './HelpCenter';
import { CARD_IDS } from './HelpCenter.constants';
import * as useUserModule from 'contexts/user';
import { OrgRole } from 'types/user.types';

describe('HelpCenter', () => {
  it('should show pmm dump and pmm longs if user is admin', () => {
    vi.spyOn(useUserModule, 'useUser').mockReturnValue({
      isLoading: false,
      user: {
        id: 1,
        isPMMAdmin: true,
        orgRole: OrgRole.Admin,
        isAuthorized: true,
      },
    });

    render(<HelpCenter />);

    expect(
      screen.queryByTestId(`help-card-${CARD_IDS.pmmDump}`)
    ).toBeInTheDocument();
    expect(
      screen.queryByTestId(`help-card-${CARD_IDS.pmmLogs}`)
    ).toBeInTheDocument();
    expect(screen.queryAllByTestId(/^help-card-/).length).toEqual(6);
  });

  it('should not show pmm dump and pmm longs if user has no org role', () => {
    vi.spyOn(useUserModule, 'useUser').mockReturnValue({
      isLoading: false,
      user: {
        id: 1,
        isPMMAdmin: false,
        orgRole: OrgRole.None,
        isAuthorized: true,
      },
    });

    render(<HelpCenter />);

    expect(
      screen.queryByTestId(`help-card-${CARD_IDS.pmmDump}`)
    ).not.toBeInTheDocument();
    expect(
      screen.queryByTestId(`help-card-${CARD_IDS.pmmLogs}`)
    ).not.toBeInTheDocument();
    expect(screen.queryAllByTestId(/^help-card-/).length).toEqual(4);
  });

  it('should not show pmm dump and pmm longs if user is viewer', () => {
    vi.spyOn(useUserModule, 'useUser').mockReturnValue({
      isLoading: false,
      user: {
        id: 1,
        isPMMAdmin: false,
        orgRole: OrgRole.Viewer,
        isAuthorized: true,
      },
    });

    render(<HelpCenter />);

    expect(
      screen.queryByTestId(`help-card-${CARD_IDS.pmmDump}`)
    ).not.toBeInTheDocument();
    expect(
      screen.queryByTestId(`help-card-${CARD_IDS.pmmLogs}`)
    ).not.toBeInTheDocument();
    expect(screen.queryAllByTestId(/^help-card-/).length).toEqual(4);
  });

  it('should not show pmm dump and pmm longs if user is editor', () => {
    vi.spyOn(useUserModule, 'useUser').mockReturnValue({
      isLoading: false,
      user: {
        id: 1,
        isPMMAdmin: false,
        orgRole: OrgRole.Editor,
        isAuthorized: true,
      },
    });

    render(<HelpCenter />);

    expect(
      screen.queryByTestId(`help-card-${CARD_IDS.pmmDump}`)
    ).not.toBeInTheDocument();
    expect(
      screen.queryByTestId(`help-card-${CARD_IDS.pmmLogs}`)
    ).not.toBeInTheDocument();
    expect(screen.queryAllByTestId(/^help-card-/).length).toEqual(4);
  });
});

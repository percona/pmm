import { render, screen } from '@testing-library/react';
import { HelpCenter } from './HelpCenter';
import { cardIds } from './HelpCenter.constants';
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
      screen.queryByTestId(`help-card-${cardIds.pmmDump}`)
    ).toBeInTheDocument();
    expect(
      screen.queryByTestId(`help-card-${cardIds.pmmLogs}`)
    ).toBeInTheDocument();
    expect(screen.queryAllByTestId(/^help-card-/).length).toEqual(6);
  });

  it('should not show pmm dump and pmm longs if user is not admin', () => {
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
      screen.queryByTestId(`help-card-${cardIds.pmmDump}`)
    ).not.toBeInTheDocument();
    expect(
      screen.queryByTestId(`help-card-${cardIds.pmmLogs}`)
    ).not.toBeInTheDocument();
    expect(screen.queryAllByTestId(/^help-card-/).length).toEqual(4);
  });
});

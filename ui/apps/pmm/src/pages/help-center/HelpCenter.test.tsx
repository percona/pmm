import { render, screen } from '@testing-library/react';
import { HelpCenter } from './HelpCenter';
import { CARD_IDS } from './HelpCenter.constants';
import * as useUserModule from 'contexts/user';
import { OrgRole, User } from 'types/user.types';
import { MemoryRouter } from 'react-router-dom';

const getUser = (user: Partial<User> = {}): User => ({
  id: 1,
  isPMMAdmin: true,
  orgRole: OrgRole.Admin,
  isAuthorized: true,
  name: 'admin',
  login: 'admin',
  orgId: 1,
  isViewer: true,
  isEditor: true,
  orgs: [],
  ...user,
});

const renderHelpCenter = () =>
  render(
    <MemoryRouter>
      <HelpCenter />
    </MemoryRouter>
  );

describe('HelpCenter', () => {
  it('should show pmm dump and pmm logs if user is admin', () => {
    vi.spyOn(useUserModule, 'useUser').mockReturnValue({
      isLoading: false,
      user: getUser(),
    });

    renderHelpCenter();

    expect(
      screen.queryByTestId(`help-card-${CARD_IDS.pmmDump}`)
    ).toBeInTheDocument();
    expect(
      screen.queryByTestId(`help-card-${CARD_IDS.pmmLogs}`)
    ).toBeInTheDocument();
    expect(screen.queryAllByTestId(/^help-card-/).length).toEqual(7);
  });

  it('should not show pmm dump and pmm logs if user has no org role', () => {
    vi.spyOn(useUserModule, 'useUser').mockReturnValue({
      isLoading: false,
      user: getUser({
        isPMMAdmin: false,
        isViewer: false,
        isEditor: false,
        orgRole: OrgRole.None,
      }),
    });

    renderHelpCenter();

    expect(
      screen.queryByTestId(`help-card-${CARD_IDS.pmmDump}`)
    ).not.toBeInTheDocument();
    expect(
      screen.queryByTestId(`help-card-${CARD_IDS.pmmLogs}`)
    ).not.toBeInTheDocument();
    expect(screen.queryAllByTestId(/^help-card-/).length).toEqual(5);
  });

  it('should not show pmm dump and pmm logs if user is viewer', () => {
    vi.spyOn(useUserModule, 'useUser').mockReturnValue({
      isLoading: false,
      user: getUser({
        isViewer: true,
        isEditor: false,
        isPMMAdmin: false,
        orgRole: OrgRole.Viewer,
      }),
    });

    renderHelpCenter();

    expect(
      screen.queryByTestId(`help-card-${CARD_IDS.pmmDump}`)
    ).not.toBeInTheDocument();
    expect(
      screen.queryByTestId(`help-card-${CARD_IDS.pmmLogs}`)
    ).not.toBeInTheDocument();
    expect(screen.queryAllByTestId(/^help-card-/).length).toEqual(5);
  });

  it('should not show pmm dump and pmm logs if user is editor', () => {
    vi.spyOn(useUserModule, 'useUser').mockReturnValue({
      isLoading: false,
      user: getUser({
        isViewer: true,
        isEditor: true,
        isPMMAdmin: false,
        orgRole: OrgRole.Editor,
      }),
    });

    renderHelpCenter();

    expect(
      screen.queryByTestId(`help-card-${CARD_IDS.pmmDump}`)
    ).not.toBeInTheDocument();
    expect(
      screen.queryByTestId(`help-card-${CARD_IDS.pmmLogs}`)
    ).not.toBeInTheDocument();
    expect(screen.queryAllByTestId(/^help-card-/).length).toEqual(5);
  });
});

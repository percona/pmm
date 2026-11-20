import { screen, render } from '@testing-library/react';
import { Page } from './Page';
import { TestWrapper } from 'utils/testWrapper';
import { UserContext } from 'contexts/user';
import { Messages } from './Page.messages';
import {
  TEST_USER_ADMIN,
  TEST_USER_EDITOR,
  TEST_USER_GRAFANA_ADMIN,
} from 'utils/testStubs';
import { OrgRole, type User } from 'types/user.types';
import { measurePageSurface, measureSurface } from 'utils/testUtils';

describe('Page', () => {
  it('it shows page content when authorized', () => {
    render(
      <TestWrapper>
        <UserContext.Provider
          value={{
            isLoading: false,
            user: TEST_USER_ADMIN,
          }}
        >
          <Page>
            <div>Authorized</div>
          </Page>
        </UserContext.Provider>
      </TestWrapper>
    );

    expect(screen.queryByText('Page Content')).toBeDefined();
  });

  it('keeps the document title but hides the heading with hideTitle', () => {
    render(
      <TestWrapper>
        <Page title="Hidden title" hideTitle>
          <div>Page Content</div>
        </Page>
      </TestWrapper>
    );

    expect(screen.queryByRole('heading', { name: 'Hidden title' })).toBeNull();
    expect(document.title).toContain('Hidden title');
  });

  it('it shows no access page when unauthorized', () => {
    render(
      <TestWrapper>
        <UserContext.Provider
          value={{
            isLoading: false,
            user: { ...TEST_USER_ADMIN, isAuthorized: false },
          }}
        >
          <Page>
            <div>Page Content</div>
          </Page>
        </UserContext.Provider>
      </TestWrapper>
    );

    expect(screen.queryByText('Page Content')).toBeNull();
    expect(screen.queryByText(Messages.noAcccess)).toBeDefined();
  });

  describe('roles', () => {
    const renderPage = (user: User, roles: OrgRole[]) =>
      render(
        <TestWrapper>
          <UserContext.Provider value={{ isLoading: false, user }}>
            <Page roles={roles}>
              <div>Page Content</div>
            </Page>
          </UserContext.Provider>
        </TestWrapper>
      );

    it.each([
      ['an org Admin', TEST_USER_ADMIN],
      ['a Grafana server admin', TEST_USER_GRAFANA_ADMIN],
    ])('lets %s into an Admin page', (_, user) => {
      renderPage(user, [OrgRole.Admin]);

      expect(screen.getByText('Page Content')).toBeInTheDocument();
    });

    it('keeps an Editor out of an Admin page', () => {
      renderPage(TEST_USER_EDITOR, [OrgRole.Admin]);

      expect(screen.queryByText('Page Content')).not.toBeInTheDocument();
      expect(screen.getByText(Messages.noAcccess)).toBeInTheDocument();
    });

    it('lets a Grafana server admin into a page for other roles too', () => {
      renderPage(TEST_USER_GRAFANA_ADMIN, [OrgRole.Editor]);

      expect(screen.getByText('Page Content')).toBeInTheDocument();
    });
  });

  describe('surface', () => {
    it('renders on the paper surface when a page specifies none', () => {
      const stage = measurePageSurface('canvas');
      const paper = measurePageSurface('paper');

      // Guards the assertion below: it only means anything while the two
      // surfaces actually differ.
      expect(paper).not.toBe(stage);

      const unspecified = measureSurface(() =>
        render(
          <TestWrapper>
            <Page>
              <div>Page Content</div>
            </Page>
          </TestWrapper>
        )
      );

      expect(unspecified).toBe(paper);
    });

    it('renders on the canvas surface when a page opts out', () => {
      const paper = measurePageSurface('paper');

      const optedOut = measureSurface(() =>
        render(
          <TestWrapper>
            <Page surface="canvas">
              <div>Page Content</div>
            </Page>
          </TestWrapper>
        )
      );

      expect(optedOut).not.toBe(paper);
      expect(optedOut).toBe(measurePageSurface('canvas'));
    });
  });
});

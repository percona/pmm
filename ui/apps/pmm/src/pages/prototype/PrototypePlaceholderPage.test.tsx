import { render, screen } from '@testing-library/react';
import { Route, Routes } from 'react-router-dom';
import { NavigationContext } from 'contexts/navigation';
import { NavItem } from 'types/navigation.types';
import { TestWrapper } from 'utils/testWrapper';
import { PrototypePlaceholderPage } from './PrototypePlaceholderPage';
import { Messages } from './PrototypePlaceholderPage.messages';

const renderPage = (path: string, navTree: NavItem[] = []) =>
  render(
    <TestWrapper routerProps={{ initialEntries: [path] }}>
      <NavigationContext.Provider
        value={{
          navTree,
          navOpen: true,
          setNavOpen: () => {},
          starredApps: [],
          toggleStarredApp: () => {},
        }}
      >
        <Routes>
          <Route
            path="/prototype/:slug"
            element={<PrototypePlaceholderPage />}
          />
        </Routes>
      </NavigationContext.Provider>
    </TestWrapper>
  );

describe('PrototypePlaceholderPage', () => {
  it('titles the page after the entry that links here', () => {
    renderPage('/prototype/add-service', [
      { id: 'action', text: 'Add a service', url: '/prototype/add-service' },
    ]);

    expect(
      screen.getByRole('heading', { name: 'Add a service' })
    ).toBeInTheDocument();
  });

  it('leaves a planned app’s name to the app header above it', () => {
    renderPage('/prototype/schema-changes', [
      { id: 'app', text: 'Schema changes', url: '/prototype/schema-changes' },
    ]);

    expect(screen.queryByRole('heading')).not.toBeInTheDocument();
    expect(document.title).toContain('Schema changes');
  });

  it('falls back to the slug for an action with no entry of its own', () => {
    renderPage('/prototype/customize-navigation');

    expect(
      screen.getByRole('heading', { name: 'Customize navigation' })
    ).toBeInTheDocument();
  });

  it.each([
    'data-archiving',
    'postgresql-backups',
    'replication-checksums',
    'schema-changes',
  ])('says %s is a PMM Extensions app this build leaves out', (slug) => {
    renderPage(`/prototype/${slug}`);

    expect(screen.getByText(Messages.notInThisBuild)).toBeInTheDocument();
  });

  it.each(['proxysql-manager', 'valkey-backups', 'all-apps'])(
    'says nothing about the build for %s',
    (slug) => {
      renderPage(`/prototype/${slug}`);

      expect(screen.getByText(Messages.description)).toBeInTheDocument();
      expect(
        screen.queryByText(Messages.notInThisBuild)
      ).not.toBeInTheDocument();
    }
  );
});

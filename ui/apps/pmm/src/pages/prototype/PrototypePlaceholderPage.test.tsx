import { render, screen } from '@testing-library/react';
import { Route, Routes } from 'react-router-dom';
import { NavigationContext } from 'contexts/navigation';
import { NavItem } from 'types/navigation.types';
import { TestWrapper } from 'utils/testWrapper';
import { PrototypePlaceholderPage } from './PrototypePlaceholderPage';

const renderPage = (path: string, navTree: NavItem[] = []) =>
  render(
    <TestWrapper routerProps={{ initialEntries: [path] }}>
      <NavigationContext.Provider
        value={{ navTree, navOpen: true, setNavOpen: () => {} }}
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
    renderPage('/prototype/schema-changes', [
      { id: 'app', text: 'Schema changes', url: '/prototype/schema-changes' },
    ]);

    expect(
      screen.getByRole('heading', { name: 'Schema changes' })
    ).toBeInTheDocument();
  });

  it('falls back to the slug for an action with no entry of its own', () => {
    renderPage('/prototype/customize-navigation');

    expect(
      screen.getByRole('heading', { name: 'Customize navigation' })
    ).toBeInTheDocument();
  });
});

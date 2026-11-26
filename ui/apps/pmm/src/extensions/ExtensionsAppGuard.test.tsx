import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { useExtensionsApps } from 'hooks/api/useExtensionsApps';
import { ExtensionsAppGuard } from './ExtensionsAppGuard';
import {
  DEFAULT_EXTENSIONS_APP_ICON,
  type ExtensionsApp,
} from './extensionsApps';

vi.mock('hooks/api/useExtensionsApps', () => ({
  useExtensionsApps: vi.fn(),
}));

const extensionsApps = vi.mocked(useExtensionsApps);

const PG_BACKUPS: ExtensionsApp = {
  key: 'pg_backups',
  slug: 'pg-backups',
  label: 'PostgreSQL Backups',
  icon: DEFAULT_EXTENSIONS_APP_ICON,
  order: 10,
  enabled: true,
  customUi: false,
  blockedBy: [],
};

const ATW: ExtensionsApp = {
  ...PG_BACKUPS,
  key: 'atw',
  slug: 'atw',
  label: 'Support diagnostics',
  customUi: true,
};

const stubList = (result: Partial<ReturnType<typeof useExtensionsApps>>) => {
  const refetch = vi.fn();
  extensionsApps.mockReturnValue({
    data: undefined,
    isError: false,
    refetch,
    ...result,
  } as ReturnType<typeof useExtensionsApps>);
  return refetch;
};

const APP = <div data-testid="app" />;

const renderGeneric = (path = '/extensions/pg-backups') =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route
          path="/extensions/:appSlug/*"
          element={<ExtensionsAppGuard>{APP}</ExtensionsAppGuard>}
        />
      </Routes>
    </MemoryRouter>
  );

const renderDedicated = () =>
  render(
    <MemoryRouter initialEntries={['/extensions/atw/incidents']}>
      <Routes>
        <Route
          path="/extensions/atw/*"
          element={<ExtensionsAppGuard appKey="atw">{APP}</ExtensionsAppGuard>}
        />
      </Routes>
    </MemoryRouter>
  );

describe('ExtensionsAppGuard', () => {
  describe('on the generic route', () => {
    it('renders a listed, enabled app', () => {
      stubList({ data: [PG_BACKUPS] });

      renderGeneric('/extensions/pg-backups/backups/123');

      expect(screen.getByTestId('app')).toBeInTheDocument();
    });

    it('waits for the list before deciding', () => {
      stubList({ data: undefined });

      renderGeneric();

      expect(screen.getByRole('progressbar')).toBeInTheDocument();
      expect(screen.queryByTestId('app')).not.toBeInTheDocument();
    });

    it('offers a retry when the list cannot be loaded', () => {
      const refetch = stubList({ isError: true });

      renderGeneric();
      screen.getByRole('button', { name: 'Try again' }).click();

      expect(
        screen.getByTestId('extensions-app-load-failed')
      ).toBeInTheDocument();
      expect(refetch).toHaveBeenCalledOnce();
      expect(screen.queryByTestId('app')).not.toBeInTheDocument();
    });

    it('keeps rendering the app when only a refetch failed', () => {
      stubList({ data: [PG_BACKUPS], isError: true });

      renderGeneric();

      expect(screen.getByTestId('app')).toBeInTheDocument();
    });

    it('says so for a slug no app has', () => {
      stubList({ data: [PG_BACKUPS] });

      renderGeneric('/extensions/no-such-app');

      expect(
        screen.getByTestId('extensions-app-not-found')
      ).toBeInTheDocument();
      expect(screen.queryByTestId('app')).not.toBeInTheDocument();
    });

    it('does not turn the slug back into a key', () => {
      stubList({ data: [{ ...PG_BACKUPS, slug: 'postgresql-backups' }] });

      renderGeneric('/extensions/pg-backups');

      expect(
        screen.getByTestId('extensions-app-not-found')
      ).toBeInTheDocument();
    });

    it('does not render an app that ships a custom page', () => {
      stubList({ data: [{ ...PG_BACKUPS, customUi: true }] });

      renderGeneric();

      expect(
        screen.getByTestId('extensions-app-not-found')
      ).toBeInTheDocument();
      expect(screen.queryByTestId('app')).not.toBeInTheDocument();
    });

    it('says a disabled app is disabled', () => {
      stubList({ data: [{ ...PG_BACKUPS, enabled: false }] });

      renderGeneric();

      expect(screen.getByTestId('extensions-app-disabled')).toHaveTextContent(
        'PostgreSQL Backups is disabled'
      );
      expect(screen.getByTestId('extensions-app-disabled')).toHaveTextContent(
        'An administrator has disabled this app.'
      );
      expect(screen.queryByTestId('app')).not.toBeInTheDocument();
    });

    it('names the disabled apps that keep an app disabled', () => {
      stubList({
        data: [{ ...PG_BACKUPS, enabled: false, blockedBy: ['Inventory'] }],
      });

      renderGeneric();

      expect(screen.getByTestId('extensions-app-disabled')).toHaveTextContent(
        'This app needs Inventory, which is disabled.'
      );
    });
  });

  describe('on a dedicated route', () => {
    it('renders its enabled app, custom page included', () => {
      stubList({ data: [ATW, PG_BACKUPS] });

      renderDedicated();

      expect(screen.getByTestId('app')).toBeInTheDocument();
    });

    it('says its app is disabled instead of mounting it', () => {
      stubList({ data: [{ ...ATW, enabled: false }] });

      renderDedicated();

      expect(screen.getByTestId('extensions-app-disabled')).toHaveTextContent(
        'Support diagnostics is disabled'
      );
      expect(screen.queryByTestId('app')).not.toBeInTheDocument();
    });

    it('waits for the list before deciding', () => {
      stubList({ data: undefined });

      renderDedicated();

      expect(screen.getByRole('progressbar')).toBeInTheDocument();
      expect(screen.queryByTestId('app')).not.toBeInTheDocument();
    });

    it('renders its app when the list cannot be loaded', () => {
      stubList({ isError: true });

      renderDedicated();

      expect(screen.getByTestId('app')).toBeInTheDocument();
      expect(
        screen.queryByTestId('extensions-app-load-failed')
      ).not.toBeInTheDocument();
    });

    it('renders its app when the list does not include it', () => {
      stubList({ data: [PG_BACKUPS] });

      renderDedicated();

      expect(screen.getByTestId('app')).toBeInTheDocument();
    });
  });
});

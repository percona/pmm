import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { SchemaDrivenPlugin } from '@pmm-extensions/framework';
import { useExtensionsApps } from 'hooks/api/useExtensionsApps';
import { ExtensionsAppRoute } from './ExtensionsAppRoute';
import {
  DEFAULT_EXTENSIONS_APP_ICON,
  type ExtensionsApp,
} from './extensionsApps';

vi.mock('@pmm-extensions/framework', () => ({
  SchemaDrivenPlugin: vi.fn(() => <div data-testid="schema-driven-plugin" />),
}));

vi.mock('hooks/api/useExtensionsApps', () => ({
  useExtensionsApps: vi.fn(),
}));

const extensionsApps = vi.mocked(useExtensionsApps);
const plugin = vi.mocked(SchemaDrivenPlugin);

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

const renderRoute = (path = '/extensions/pg-backups') =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/extensions/:appSlug/*" element={<ExtensionsAppRoute />} />
      </Routes>
    </MemoryRouter>
  );

beforeEach(() => {
  plugin.mockClear();
});

describe('ExtensionsAppRoute', () => {
  it('renders a listed app through the schema-driven plugin', () => {
    stubList({ data: [PG_BACKUPS] });

    renderRoute('/extensions/pg-backups/backups/123');

    expect(screen.getByTestId('schema-driven-plugin')).toBeInTheDocument();
    expect(plugin.mock.calls[0][0]).toMatchObject({
      pluginName: 'pg_backups',
      routeBase: '/extensions/pg-backups',
    });
  });

  it('waits for the list before deciding', () => {
    stubList({ data: undefined });

    renderRoute();

    expect(screen.getByRole('progressbar')).toBeInTheDocument();
    expect(plugin).not.toHaveBeenCalled();
  });

  it('offers a retry when the list cannot be loaded', () => {
    const refetch = stubList({ isError: true });

    renderRoute();
    screen.getByRole('button', { name: 'Try again' }).click();

    expect(
      screen.getByTestId('extensions-app-load-failed')
    ).toBeInTheDocument();
    expect(refetch).toHaveBeenCalled();
    expect(plugin).not.toHaveBeenCalled();
  });

  it('keeps rendering the app when only a refetch failed', () => {
    stubList({ data: [PG_BACKUPS], isError: true });

    renderRoute();

    expect(screen.getByTestId('schema-driven-plugin')).toBeInTheDocument();
  });

  it('says so for a slug no app has', () => {
    stubList({ data: [PG_BACKUPS] });

    renderRoute('/extensions/no-such-app');

    expect(screen.getByTestId('extensions-app-not-found')).toBeInTheDocument();
    expect(plugin).not.toHaveBeenCalled();
  });

  it('does not turn the slug back into a key', () => {
    stubList({ data: [{ ...PG_BACKUPS, slug: 'postgresql-backups' }] });

    renderRoute('/extensions/pg-backups');

    expect(screen.getByTestId('extensions-app-not-found')).toBeInTheDocument();
  });

  it('does not render an app that ships a custom page', () => {
    stubList({ data: [{ ...PG_BACKUPS, customUi: true }] });

    renderRoute();

    expect(screen.getByTestId('extensions-app-not-found')).toBeInTheDocument();
    expect(plugin).not.toHaveBeenCalled();
  });

  it('says a disabled app is disabled', () => {
    stubList({ data: [{ ...PG_BACKUPS, enabled: false }] });

    renderRoute();

    expect(screen.getByTestId('extensions-app-disabled')).toHaveTextContent(
      'PostgreSQL Backups is disabled'
    );
    expect(screen.getByTestId('extensions-app-disabled')).toHaveTextContent(
      'An administrator has disabled this app.'
    );
    expect(plugin).not.toHaveBeenCalled();
  });

  it('names the disabled apps that keep an app disabled', () => {
    stubList({
      data: [{ ...PG_BACKUPS, enabled: false, blockedBy: ['Inventory'] }],
    });

    renderRoute();

    expect(screen.getByTestId('extensions-app-disabled')).toHaveTextContent(
      'This app needs Inventory, which is disabled.'
    );
  });
});

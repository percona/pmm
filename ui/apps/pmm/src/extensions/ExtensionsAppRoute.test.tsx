import { render } from '@testing-library/react';
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

const renderRoute = (path: string) =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/extensions/:appSlug/*" element={<ExtensionsAppRoute />} />
      </Routes>
    </MemoryRouter>
  );

beforeEach(() => {
  plugin.mockClear();
  vi.mocked(useExtensionsApps).mockReturnValue({
    data: [PG_BACKUPS],
  } as ReturnType<typeof useExtensionsApps>);
});

describe('ExtensionsAppRoute', () => {
  it('renders the app the slug names through the schema-driven plugin', () => {
    renderRoute('/extensions/pg-backups/backups/123');

    expect(plugin).toHaveBeenCalledOnce();
    expect(plugin.mock.calls[0][0]).toMatchObject({
      pluginName: 'pg_backups',
      routeBase: '/extensions/pg-backups',
    });
  });

  it('renders nothing for a slug no app has', () => {
    const { container } = renderRoute('/extensions/no-such-app');

    expect(container).toBeEmptyDOMElement();
    expect(plugin).not.toHaveBeenCalled();
  });
});

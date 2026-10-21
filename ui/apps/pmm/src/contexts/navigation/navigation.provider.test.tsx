import { render, screen } from '@testing-library/react';
import { FC } from 'react';
import { Settings } from 'types/settings.types';
import { TEST_USER_VIEWER } from 'utils/testStubs';
import { wrapWithQueryProvider, wrapWithUserProvider } from 'utils/testUtils';
import { useNavigation } from './navigation.hooks';
import { NavigationProvider } from './navigation.provider';

vi.mock('hooks/api/useServices', () => ({
  useServiceTypes: () => ({ data: { serviceTypes: [] } }),
}));
vi.mock('hooks/api/useAdvisors', () => ({ useAdvisors: () => ({}) }));
vi.mock('hooks/api/useFolders', () => ({ useFolders: () => ({ data: [] }) }));
vi.mock('hooks/api/useHA', () => ({
  useHaInfo: () => ({ data: { enabled: false } }),
}));
vi.mock('contexts/updates', () => ({ useUpdates: () => ({}) }));
vi.mock('contexts/settings', () => ({
  useSettings: () => ({
    settings: {
      frontend: {
        exploreEnabled: false,
        unifiedAlertingEnabled: false,
        apps: {},
      },
    } as unknown as Settings,
  }),
}));

const NavIds: FC = () => {
  const { navTree } = useNavigation();
  return <div data-testid="nav-ids">{navTree.map((i) => i.id).join(',')}</div>;
};

describe('NavigationProvider', () => {
  it('shows Logs to viewers', () => {
    render(
      wrapWithQueryProvider(
        wrapWithUserProvider(
          <NavigationProvider>
            <NavIds />
          </NavigationProvider>,
          { user: TEST_USER_VIEWER }
        )
      )
    );
    expect(screen.getByTestId('nav-ids').textContent?.split(',')).toContain(
      'logs'
    );
  });
});

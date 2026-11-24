import { renderHook } from '@testing-library/react';
import type { ReactElement } from 'react';
import type { AuthContextProps } from 'contexts/auth';
import { ServiceType } from 'types/services.types';
import type { User } from 'types/user.types';
import { TEST_USER_ADMIN, TEST_USER_VIEWER } from 'utils/testStubs';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithSettings, wrapWithUpdatesProvider } from 'utils/testUtils';
import { NavigationProvider } from './navigation.provider';
import { useNavigation } from './navigation.hooks';
import {
  NAV_ALERTS,
  NAV_DIAGNOSTICS,
  NAV_HELP,
  NAV_INVENTORY,
  NAV_MONGO,
  NAV_MYSQL,
  NAV_OS,
  NAV_POSTGRESQL,
  NAV_QAN,
  NAV_REALTIME_QUERIES,
  NAV_SIGN_IN,
  NAV_VALKEY,
} from './navigation.constants';

const serviceTypes = vi.hoisted(() => ({ current: [] as string[] }));

vi.mock('hooks/api/useServices', () => ({
  useServiceTypes: () => ({ data: { serviceTypes: serviceTypes.current } }),
}));

vi.mock('hooks/api/useFolders', () => ({
  useFolders: () => ({ data: [] }),
}));

vi.mock('hooks/api/useHA', () => ({
  useHaInfo: () => ({ data: { enabled: false, nodes: [] } }),
}));

vi.mock('hooks/theme', () => ({
  useColorMode: () => ({ colorMode: 'light', toggleColorMode: () => {} }),
}));

const ANONYMOUS_AUTH: AuthContextProps = {
  isLoading: false,
  isLoggedIn: false,
  isAnonymous: true,
};

const renderNavTree = (
  user: User | null = TEST_USER_ADMIN,
  authContext?: AuthContextProps
) => {
  const { result } = renderHook(() => useNavigation(), {
    wrapper: ({ children }) => (
      <TestWrapper
        authContext={authContext}
        userContext={{ isLoading: false, user: user ?? undefined }}
      >
        {wrapWithSettings(
          wrapWithUpdatesProvider(
            <NavigationProvider>{children}</NavigationProvider>
          ) as ReactElement,
          { settings: { extensionsEnabled: true, omEnabled: true } }
        )}
      </TestWrapper>
    ),
  });

  return result.current.navTree.map((item) => item.id);
};

describe('NavigationProvider', () => {
  afterEach(() => {
    serviceTypes.current = [];
  });

  it('serves the sectioned sidebar to a signed-in admin', () => {
    const ids = renderNavTree();

    expect(ids[0]).toBe('section-technologies');
    expect(ids).toContain('section-technologies');
    expect(ids).toContain('section-administration');
    expect(ids).toContain('configuration');
    expect(ids).toContain('help');
    expect(ids).not.toContain('sign-in');
  });

  it('lists only the technologies PMM monitors', () => {
    serviceTypes.current = [ServiceType.mysql];

    const ids = renderNavTree();

    expect(ids).toContain('mysql');
    expect(ids).not.toContain('mongo');
    expect(ids).toContain('system');
  });

  it('withholds administration from a viewer', () => {
    const ids = renderNavTree(TEST_USER_VIEWER);

    expect(ids).toContain(NAV_QAN.id);
    expect(ids).not.toContain('section-administration');
    expect(ids).not.toContain('configuration');
    expect(ids).not.toContain('inventory');
  });

  describe('for an anonymous visitor', () => {
    const anonymousIds = () => renderNavTree(null, ANONYMOUS_AUTH);

    it('lists every database menu, since they cannot list service types', () => {
      expect(anonymousIds()).toEqual(
        expect.arrayContaining([
          NAV_MYSQL.id,
          NAV_MONGO.id,
          NAV_POSTGRESQL.id,
          NAV_VALKEY.id,
          NAV_OS.id,
        ])
      );
    });

    it('offers sign in and nothing that needs PMM APIs', () => {
      const ids = anonymousIds();

      expect(ids).toContain(NAV_SIGN_IN.id);
      for (const id of [
        NAV_QAN.id,
        NAV_REALTIME_QUERIES.id,
        NAV_ALERTS.id,
        NAV_DIAGNOSTICS.id,
        'apps',
        NAV_INVENTORY.id,
        NAV_HELP.id,
        'account',
      ]) {
        expect(ids).not.toContain(id);
      }
    });
  });
});

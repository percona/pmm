import { renderHook } from '@testing-library/react';
import type { ReactElement } from 'react';
import { ServiceType } from 'types/services.types';
import type { User } from 'types/user.types';
import { TEST_USER_ADMIN, TEST_USER_VIEWER } from 'utils/testStubs';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithSettings, wrapWithUpdatesProvider } from 'utils/testUtils';
import { NavigationProvider } from './navigation.provider';
import { useNavigation } from './navigation.hooks';

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

const renderNavTree = (user: User = TEST_USER_ADMIN) => {
  const { result } = renderHook(() => useNavigation(), {
    wrapper: ({ children }) => (
      <TestWrapper userContext={{ isLoading: false, user }}>
        {wrapWithSettings(
          wrapWithUpdatesProvider(
            <NavigationProvider>{children}</NavigationProvider>
          ) as ReactElement
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

    expect(ids).not.toContain('section-administration');
    expect(ids).not.toContain('configuration');
    expect(ids).not.toContain('inventory');
  });
});

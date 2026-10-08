import { renderHook } from '@testing-library/react';
import { ReactElement } from 'react';
import { User } from 'types/user.types';
import { TEST_USER_ADMIN, TEST_USER_VIEWER } from 'utils/testStubs';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithSettings, wrapWithUpdatesProvider } from 'utils/testUtils';
import { NavigationProvider } from './navigation.provider';
import { useNavigation } from './navigation.hooks';

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

  return result.current.navTree;
};

describe('NavigationProvider', () => {
  it('serves the proposed sidebar to the signed-in user', () => {
    const ids = renderNavTree().map((item) => item.id);

    expect(ids[0]).toBe('home-page');
    expect(ids).toContain('section-technologies');
    expect(ids).toContain('apps');
    expect(ids).toContain('account');
    expect(ids).not.toContain('sign-in');
  });

  it('serves a viewer the same structure as an admin', () => {
    expect(renderNavTree(TEST_USER_VIEWER).map((item) => item.id)).toEqual(
      renderNavTree().map((item) => item.id)
    );
  });
});

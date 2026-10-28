import { renderHook } from '@testing-library/react';
import type { PropsWithChildren } from 'react';
import { AuthContext } from 'contexts/auth';
import { OrgRole } from 'types/user.types';
import { TEST_USER_ADMIN } from 'utils/testStubs';
import { UserProvider } from './user.provider';
import { useUser } from './user.hooks';

const { useCurrentUser, useCurrentUserOrgs, useUserInfo, useUserPreferences } =
  vi.hoisted(() => ({
    useCurrentUser: vi.fn(),
    useCurrentUserOrgs: vi.fn(),
    useUserInfo: vi.fn(),
    useUserPreferences: vi.fn(),
  }));

vi.mock('hooks/api/useUser', () => ({
  useCurrentUser,
  useCurrentUserOrgs,
  useUserInfo,
  useUserPreferences,
}));

const loaded = <T,>(data: T) => ({ data, isLoading: false, error: null });

const mockQueries = (preferences: object = loaded({})) => {
  useCurrentUser.mockReturnValue(
    loaded({ id: 1, login: 'admin', name: 'admin', orgId: 1 })
  );
  useCurrentUserOrgs.mockReturnValue(
    loaded([{ orgId: 1, name: 'Main Org.', role: OrgRole.Admin }])
  );
  useUserInfo.mockReturnValue(loaded(TEST_USER_ADMIN.info));
  useUserPreferences.mockReturnValue(preferences);
};

const renderUser = (isLoggedIn: boolean) =>
  renderHook(() => useUser(), {
    wrapper: ({ children }: PropsWithChildren) => (
      <AuthContext.Provider value={{ isLoading: false, isLoggedIn }}>
        <UserProvider>{children}</UserProvider>
      </AuthContext.Provider>
    ),
  }).result.current;

describe('UserProvider', () => {
  it('builds the user once every query has data', () => {
    mockQueries();

    expect(renderUser(true)).toMatchObject({
      isLoading: false,
      user: { id: 1, isPMMAdmin: true },
    });
  });

  it('builds no user while logged out, even from cached data', () => {
    mockQueries();

    expect(renderUser(false).user).toBeUndefined();
  });

  it('stays loading until preferences arrive', () => {
    mockQueries({ data: undefined, isLoading: true, error: null });

    expect(renderUser(true)).toEqual({ isLoading: true, user: undefined });
  });
});

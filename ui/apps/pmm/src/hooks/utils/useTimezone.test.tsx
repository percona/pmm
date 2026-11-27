import { renderHook } from '@testing-library/react';
import type { FC, PropsWithChildren } from 'react';
import type { UserPreferences } from 'types/user.types';
import { TestWrapper } from 'utils/testWrapper';
import { TEST_USER_ADMIN } from 'utils/testStubs';
import { useTimezone } from './useTimezone';

const renderTimezone = (preferences: UserPreferences) => {
  const wrapper: FC<PropsWithChildren> = ({ children }) => (
    <TestWrapper
      userContext={{
        user: { ...TEST_USER_ADMIN, preferences },
        isLoading: false,
      }}
    >
      {children}
    </TestWrapper>
  );

  return renderHook(() => useTimezone(), { wrapper }).result.current;
};

describe('useTimezone', () => {
  it("resolves the browser preference to the browser's zone", () => {
    expect(renderTimezone({ timezone: 'browser' })).toBe(
      Intl.DateTimeFormat().resolvedOptions().timeZone
    );
  });

  it('defaults to UTC', () => {
    expect(renderTimezone({})).toBe('UTC');
  });

  it('keeps an IANA preference', () => {
    expect(renderTimezone({ timezone: 'Asia/Tokyo' })).toBe('Asia/Tokyo');
  });
});

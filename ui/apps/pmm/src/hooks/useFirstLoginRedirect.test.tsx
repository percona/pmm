import { afterEach, describe, expect, it } from 'vitest';
import { cleanup, render, screen } from '@testing-library/react';
import { MemoryRouter, useLocation } from 'react-router-dom';
import { UserContext } from 'contexts/user';
import { TEST_USER_ADMIN } from 'utils/testStubs';
import type { User } from 'types/user.types';
import { useFirstLoginRedirect } from './useFirstLoginRedirect';

const FIRST_LOGIN_KEY = `pmm-ui.first-login.user-${TEST_USER_ADMIN.id}`;

const Probe = () => {
  useFirstLoginRedirect();
  const location = useLocation();

  return <div data-testid="location">{location.pathname}</div>;
};

const renderAt = (path: string, user: User = TEST_USER_ADMIN) =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <UserContext.Provider value={{ isLoading: false, user }}>
        <Probe />
      </UserContext.Provider>
    </MemoryRouter>
  );

describe('useFirstLoginRedirect', () => {
  afterEach(() => {
    cleanup();
    localStorage.clear();
  });

  it.each(['/graph', '/graph/'])(
    'sends a first-time user from %s to the help page',
    (path) => {
      renderAt(path);

      expect(screen.getByTestId('location')).toHaveTextContent('/help');
      expect(localStorage.getItem(FIRST_LOGIN_KEY)).toBe('false');
    }
  );

  it('does not redirect a user who has already seen the welcome page', () => {
    localStorage.setItem(FIRST_LOGIN_KEY, 'false');

    renderAt('/graph');

    expect(screen.getByTestId('location')).toHaveTextContent('/graph');
  });

  it('leaves a deep link alone, so a restored return-to always wins', () => {
    renderAt('/graph/d/node-cpu');

    expect(screen.getByTestId('location')).toHaveTextContent(
      '/graph/d/node-cpu'
    );
    expect(localStorage.getItem(FIRST_LOGIN_KEY)).toBeNull();
  });

  it('does not redirect an anonymous user', () => {
    renderAt('/graph', { ...TEST_USER_ADMIN, isAnonymous: true });

    expect(screen.getByTestId('location')).toHaveTextContent('/graph');
    expect(localStorage.getItem(FIRST_LOGIN_KEY)).toBeNull();
  });
});

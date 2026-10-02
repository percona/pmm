import { afterEach, describe, expect, it } from 'vitest';
import { useState } from 'react';
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
} from '@testing-library/react';
import { MemoryRouter, useLocation, useNavigate } from 'react-router-dom';
import { UserContext } from 'contexts/user';
import { TEST_USER_ADMIN } from 'utils/testStubs';
import type { User } from 'types/user.types';
import { useFirstLoginRedirect } from './useFirstLoginRedirect';

const FIRST_LOGIN_KEY = `pmm-ui.first-login.user-${TEST_USER_ADMIN.id}`;

const Probe = () => {
  useFirstLoginRedirect();
  const location = useLocation();
  const navigate = useNavigate();

  return (
    <>
      <div data-testid="location">{location.pathname}</div>
      <button onClick={() => navigate('/graph/')}>go home</button>
      <button onClick={() => navigate('/graph/d/node-cpu')}>
        go to dashboard
      </button>
    </>
  );
};

const renderAt = (path: string, user: User = TEST_USER_ADMIN) =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <UserContext.Provider value={{ isLoading: false, user }}>
        <Probe />
      </UserContext.Provider>
    </MemoryRouter>
  );

/** Lets a test mount with no user yet, then resolve one the way UserProvider's queries do. */
const LateUser = ({ path }: { path: string }) => {
  const [user, setUser] = useState<User | undefined>(undefined);

  return (
    <MemoryRouter initialEntries={[path]}>
      <UserContext.Provider value={{ isLoading: !user, user }}>
        <Probe />
        <button onClick={() => setUser(TEST_USER_ADMIN)}>resolve user</button>
      </UserContext.Provider>
    </MemoryRouter>
  );
};

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

  it('does not fire on an in-session navigation to the Grafana home', () => {
    // Navigating to /graph/ mid-session is not a boot, so the welcome page must not fire.
    renderAt('/graph/d/node-cpu');

    act(() => {
      fireEvent.click(screen.getByRole('button', { name: 'go home' }));
    });

    expect(screen.getByTestId('location')).toHaveTextContent('/graph/');
    expect(localStorage.getItem(FIRST_LOGIN_KEY)).toBeNull();
  });

  it('does not redirect an anonymous user', () => {
    renderAt('/graph', { ...TEST_USER_ADMIN, isAnonymous: true });

    expect(screen.getByTestId('location')).toHaveTextContent('/graph');
    expect(localStorage.getItem(FIRST_LOGIN_KEY)).toBeNull();
  });

  it('does not yank the user off a page they opened while the user was still loading', () => {
    // `user` resolves after mount, by which time the user may have clicked into a dashboard.
    render(<LateUser path="/graph/" />);

    fireEvent.click(screen.getByText('go to dashboard'));
    act(() => {
      fireEvent.click(screen.getByText('resolve user'));
    });

    expect(screen.getByTestId('location').textContent).toBe(
      '/graph/d/node-cpu'
    );
  });
});

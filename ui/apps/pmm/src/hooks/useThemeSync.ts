import { useEffect, useRef } from 'react';
import { useUserPreferences } from './api/useUser';
import { useColorMode } from './theme';
import { useAuth } from 'contexts/auth';
import { ColorMode } from '@pmm/shared';

const DEFAULT_THEME: ColorMode = 'dark';

/**
 * Synchronizes user theme from Grafana API on initial load.
 * Fixes PMM-14624 where new users inherit theme from localStorage.
 *
 * Problem: /api/user returns empty theme field, and ThemeContextProvider
 * was using localStorage causing theme inheritance between users.
 *
 * Solution: Load theme from /api/user/preferences on login and apply
 * via setFromGrafana (without broadcast/persist to avoid loops).
 */
export const useThemeSync = () => {
  const auth = useAuth();
  const {
    data: preferences,
    isLoading,
    error,
  } = useUserPreferences({
    enabled: auth.isLoggedIn,
  });
  const { setFromGrafana } = useColorMode();
  const syncedRef = useRef(false);

  // Reset synced flag when user logs out
  useEffect(() => {
    if (!auth.isLoggedIn) {
      syncedRef.current = false;
    }
  }, [auth.isLoggedIn]);

  useEffect(() => {
    if (!auth.isLoggedIn || isLoading || syncedRef.current || !preferences) {
      return;
    }

    if (error) {
      // eslint-disable-next-line no-console
      console.error('[useThemeSync] Failed to load user preferences:', error);
      return;
    }

    const themeToApply = preferences.theme || DEFAULT_THEME;

    // Apply theme from preferences
    setFromGrafana(themeToApply)
      .then(() => {
        syncedRef.current = true;
      })
      .catch((err: unknown) => {
        // eslint-disable-next-line no-console
        console.error('[useThemeSync] Failed to apply theme:', err);
      });
  }, [preferences, isLoading, error, setFromGrafana, auth.isLoggedIn]);
};

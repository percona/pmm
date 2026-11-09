import { useEffect, useRef } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { useUser } from 'contexts/user';
import { GRAFANA_HOME_PATHS, PMM_HELP_PATH } from 'lib/constants';
import { safeLocalStorage } from 'utils/storage.utils';

/** Same key the compat plugin used, so users who saw the welcome page before an upgrade don't again. */
const getFirstLoginKey = (userId: number) =>
  `pmm-ui.first-login.user-${userId}`;

/**
 * On a user's first login, send them to the help page so the welcome modal shows. Booting on the
 * Grafana home path is also the signal that no deep link was restored, since those are never
 * stored as a return-to target (see isRestorableReturnTo).
 *
 * Boot and live pathname guard different things: boot stops the welcome page hijacking a later
 * in-session navigation back to /graph/, live stops it firing after the user has moved on - `user`
 * resolves well after mount, and the Grafana iframe is interactive meanwhile.
 */
export const useFirstLoginRedirect = () => {
  const { user } = useUser();
  const location = useLocation();
  const navigate = useNavigate();
  const bootedAtHomeRef = useRef(
    GRAFANA_HOME_PATHS.includes(location.pathname)
  );

  useEffect(() => {
    if (
      !bootedAtHomeRef.current ||
      !GRAFANA_HOME_PATHS.includes(location.pathname) ||
      !user ||
      user.isAnonymous
    ) {
      return;
    }

    const key = getFirstLoginKey(user.id);

    if (safeLocalStorage.getItem(key) === 'false') {
      return;
    }

    safeLocalStorage.setItem(key, 'false');
    navigate(PMM_HELP_PATH, { replace: true });
  }, [location.pathname, navigate, user]);
};

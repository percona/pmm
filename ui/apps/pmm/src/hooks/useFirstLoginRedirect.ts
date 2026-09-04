import { useEffect, useRef } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { useUser } from 'contexts/user';
import { GRAFANA_SUB_PATH, PMM_HELP_PATH } from 'lib/constants';

/**
 * Same key and value the compat plugin used, so users who already saw the welcome page before an
 * upgrade are not shown it again.
 */
const getFirstLoginKey = (userId: number) =>
  `pmm-ui.first-login.user-${userId}`;

const HOME_PATHS = [GRAFANA_SUB_PATH, `${GRAFANA_SUB_PATH}/`];

/**
 * On a user's first login, send them to the help page so the welcome modal shows. This used to live
 * in the compat plugin, which could only see it when Grafana booted at top level on its home path —
 * nginx now redirects /graph/ into the shell, so the decision belongs here.
 *
 * Landing on the Grafana home path is also the signal that no deep link was restored: a return-to
 * target would already have navigated us elsewhere, and the home paths are never stored as one
 * (see isRestorableReturnTo), so an explicit deep link always wins over the welcome page.
 */
export const useFirstLoginRedirect = () => {
  const { user } = useUser();
  const location = useLocation();
  const navigate = useNavigate();
  const handledRef = useRef(false);

  useEffect(() => {
    if (handledRef.current || !user || user.isAnonymous) {
      return;
    }

    if (!HOME_PATHS.includes(location.pathname)) {
      return;
    }

    handledRef.current = true;

    const key = getFirstLoginKey(user.id);

    if (localStorage.getItem(key) === 'false') {
      return;
    }

    localStorage.setItem(key, 'false');
    navigate(PMM_HELP_PATH, { replace: true });
  }, [location.pathname, navigate, user]);
};

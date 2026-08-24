import { PMM_LOGIN_URL } from 'lib/constants';
import { saveReturnTo } from './auth.returnTo';

export const getSessionExpiry = () => {
  const expiryCookie = getSessionExpiryCookie();

  if (!expiryCookie) {
    return 0;
  }

  const expiresStr = expiryCookie.split('=')[1];

  if (!expiresStr) {
    return 0;
  }

  return parseInt(expiresStr, 10);
};

// based on grafana
export const getRefetchInterval = () => {
  // get the time token is going to expire
  const expires = getSessionExpiry();

  // because this job is scheduled for every tab we have open that shares a session we try
  // to distribute the scheduling of the job. For now this can be between 1 and 20 seconds
  const expiresWithDistribution =
    expires - Math.floor(Math.random() * (20 - 1) + 1);

  return expiresWithDistribution * 1000 - Date.now();
};

export const getSessionExpiryCookie = () =>
  document.cookie
    .split('; ')
    .find((row) => row.startsWith('grafana_session_expiry='));

export const redirectToLogin = () => {
  // Single choke point for every bounce to login, so the requested URL is always remembered.
  // Grafana's own ?redirectTo= is deliberately not used: it is replayed client-side on a later
  // page load, which under the /pmm-ui redirect happens inside the Grafana iframe and would leave
  // the shell's URL behind.
  saveReturnTo();
  window.location.replace(PMM_LOGIN_URL);
};

import { useUser } from 'contexts/user/user.hooks';

// Grafana stores "browser" for users who follow their browser's zone
const BROWSER_TIMEZONE = 'browser';

export const useTimezone = () => {
  const { user } = useUser();
  const timezone = user?.preferences?.timezone;

  if (timezone === BROWSER_TIMEZONE) {
    return Intl.DateTimeFormat().resolvedOptions().timeZone;
  }

  return timezone || 'UTC';
};

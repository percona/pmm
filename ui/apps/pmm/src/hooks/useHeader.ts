import { findAppPage } from 'contexts/navigation/navigation.proposal';
import { PMM_NEW_NAV_GRAFANA_PATH, PMM_NEW_NAV_PATH } from 'lib/constants';
import { useLocation } from 'react-router-dom';

// Every app gets the app header above its page, in the shell, so Grafana pages
// have it too: framed ones and the Real-time pages sit 16px in, others at the
// full-width page padding.
export const useHeader = () => {
  const { pathname } = useLocation();
  const page = findAppPage(pathname);
  const framed =
    pathname.startsWith(`${PMM_NEW_NAV_GRAFANA_PATH}/`) ||
    pathname.startsWith(`${PMM_NEW_NAV_PATH}/rta`);

  return {
    visible: !!page,
    page,
    inset: framed ? ('frame' as const) : ('page' as const),
  };
};

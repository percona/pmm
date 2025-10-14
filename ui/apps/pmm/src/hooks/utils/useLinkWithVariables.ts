import {
  DashboardVariablesMessage,
  DashboardVariablesResult,
} from '@pmm/shared';
import {
  GRAFANA_SUB_PATH,
  PMM_NEW_NAV_GRAFANA_PATH,
  PMM_NEW_NAV_PATH,
} from 'lib/constants';
import messenger from 'lib/messenger';
import { constructUrl } from 'utils/link.utils';
import { useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';

export const useLinkWithVariables = (url?: string) => {
  const [link, setLink] = useState(url);
  const location = useLocation();

  const enhanceWithVariables = async (url: string) => {
    const msg: DashboardVariablesMessage = {
      type: 'DASHBOARD_VARIABLES',
      payload: {
        url: url.replace(PMM_NEW_NAV_GRAFANA_PATH, ''),
      },
    };
    try {
      const res: DashboardVariablesResult =
        await messenger.sendMessageWithResult(msg);
      return res.url.startsWith(GRAFANA_SUB_PATH)
        ? PMM_NEW_NAV_PATH + res.url
        : PMM_NEW_NAV_GRAFANA_PATH + res.url;
    } catch {
      return url;
    }
  };

  useEffect(() => {
    if (!url || !url.includes('/d/')) {
      return;
    }

    // if it's the current dashboards just update url
    if (url.includes(location.pathname)) {
      setLink(constructUrl(location));
    } else {
      enhanceWithVariables(url).then(setLink);
    }
  }, [url, location]);

  return link;
};

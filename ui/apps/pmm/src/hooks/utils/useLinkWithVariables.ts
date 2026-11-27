import type {
  DashboardVariablesMessage,
  DashboardVariablesResult,
} from '@pmm/shared';
import {
  GRAFANA_SUB_PATH,
  PMM_NEW_NAV_GRAFANA_PATH,
  PMM_NEW_NAV_PATH,
} from 'lib/constants';
import messenger from 'lib/messenger';
import { constructUrl, keepTimeRange } from 'utils/link.utils';
import { useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { useGrafana } from 'contexts/grafana';

export const useLinkWithVariables = (url?: string) => {
  // Tagged with the URL it was resolved for, so a changed URL is followed at once.
  const [resolved, setResolved] = useState<{ url: string; link: string }>();
  const link = resolved && resolved.url === url ? resolved.link : url;
  const location = useLocation();
  const { grafanaReady } = useGrafana();

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
      setResolved({ url, link: constructUrl(location) });
    } else if (grafanaReady) {
      // Grafana can answer out of order, so drop an answer for a superseded URL
      let stale = false;
      enhanceWithVariables(url).then((expanded) => {
        if (!stale) {
          setResolved({ url, link: expanded });
        }
      });
      return () => {
        stale = true;
      };
    }
  }, [url, location, grafanaReady]);

  // Grafana expands the link only once the dashboard has loaded, so a click before that keeps the time range
  return link && keepTimeRange(link, location);
};

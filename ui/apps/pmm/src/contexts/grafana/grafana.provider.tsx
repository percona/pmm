import { FC, PropsWithChildren, useEffect, useRef, useState } from 'react';
import { GrafanaContext } from './grafana.context';
import { useLocation, useNavigate } from 'react-router';
import { PMM_NEW_NAV_GRAFANA_PATH, PMM_NEW_NAV_PATH } from 'lib/constants';
import { LocationChangeMessage } from '@pmm/shared';
import messenger from 'lib/messenger';
import { getLocationUrl } from './grafana.utils';

export const GrafanaProvider: FC<PropsWithChildren> = ({ children }) => {
  const location = useLocation();
  const src = location.pathname.replace(PMM_NEW_NAV_PATH, '');
  const isGrafanaPage = src.startsWith('/graph');
  const [isLoaded, setIsloaded] = useState(false);
  const frameRef = useRef<HTMLIFrameElement>();
  const navigate = useNavigate();

  useEffect(() => {
    if (isGrafanaPage) {
      setIsloaded(true);
    }
  }, [isGrafanaPage]);

  useEffect(() => {
    // don't send location change if it's coming from within grafana
    if (location.pathname.includes('/graph') && !location.state?.fromGrafana) {
      messenger.sendMessage({
        type: 'LOCATION_CHANGE',
        data: {
          ...location,
          pathname: location.pathname.replace(PMM_NEW_NAV_GRAFANA_PATH, ''),
        },
      });
    }
  }, [location]);

  useEffect(() => {
    if (!isLoaded) {
      return;
    }

    messenger.setWindow(frameRef.current?.contentWindow!, '#grafana-iframe');
    messenger.register();

    messenger.addListener({
      type: 'MESSENGER_READY',
      onMessage: console.log,
    });

    messenger.addListener({
      type: 'LOCATION_CHANGE',
      onMessage: ({ data: location }: LocationChangeMessage) => {
        if (!location) {
          return;
        }

        navigate(getLocationUrl(location), {
          state: { fromGrafana: true },
        });
      },
    });
  }, [isLoaded, navigate]);

  return (
    <GrafanaContext.Provider
      value={{
        frameRef,
        isFrameLoaded: isLoaded,
        isOnGrafanaPage: isGrafanaPage,
      }}
    >
      {children}
    </GrafanaContext.Provider>
  );
};

import { FC, PropsWithChildren, useEffect, useRef, useState } from 'react';
import { GrafanaContext } from './grafana.context';
import { useLocation, useNavigate, useNavigationType } from 'react-router';
import { PMM_NEW_NAV_GRAFANA_PATH, PMM_NEW_NAV_PATH } from 'lib/constants';
import { DocumentTitleUpdateMessage, LocationChangeMessage } from '@pmm/shared';
import messenger from 'lib/messenger';
import { getLocationUrl } from './grafana.utils';
import { updateDocumentTitle } from 'lib/utils/document.utils';

export const GrafanaProvider: FC<PropsWithChildren> = ({ children }) => {
  const navigationType = useNavigationType();
  const location = useLocation();
  const src = location.pathname.replace(PMM_NEW_NAV_PATH, '');
  const isGrafanaPage = src.startsWith('/graph');
  const [isLoaded, setIsloaded] = useState(false);
  const frameRef = useRef<HTMLIFrameElement>(null);
  const navigate = useNavigate();

  useEffect(() => {
    if (isGrafanaPage) {
      setIsloaded(true);
    }
  }, [isGrafanaPage]);

  useEffect(() => {
    // don't send location change if it's coming from within grafana or is POP type
    if (
      !location.pathname.includes('/graph') ||
      (location.state?.fromGrafana && navigationType !== 'POP')
    ) {
      return;
    }

    messenger.sendMessage({
      type: 'LOCATION_CHANGE',
      payload: {
        ...location,
        pathname: location.pathname.replace(PMM_NEW_NAV_GRAFANA_PATH, ''),
        action: navigationType,
      },
    });
  }, [location, navigationType]);

  useEffect(() => {
    if (!isLoaded) {
      return;
    }

    messenger
      .setTargetWindow(frameRef.current?.contentWindow!, '#grafana-iframe')
      .register();

    messenger.addListener({
      type: 'LOCATION_CHANGE',
      onMessage: ({ payload: location }: LocationChangeMessage) => {
        if (!location || location.action === 'POP') {
          return;
        }

        navigate(getLocationUrl(location), {
          state: { fromGrafana: true },
          replace: true,
        });
      },
    });

    messenger.addListener({
      type: 'DOCUMENT_TITLE_CHANGE',
      onMessage: ({ payload }: DocumentTitleUpdateMessage) => {
        if (!payload) {
          return;
        }

        updateDocumentTitle(payload.title);
      },
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isLoaded]);

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

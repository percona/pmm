import {
  FC,
  PropsWithChildren,
  useEffect,
  useRef,
  useState,
  useRef as useRef2,
} from 'react';
import { GrafanaContext } from './grafana.context';
import { useLocation, useNavigate, useNavigationType } from 'react-router';
import {
  GRAFANA_SUB_PATH,
  PMM_NEW_NAV_GRAFANA_PATH,
  PMM_NEW_NAV_PATH,
} from 'lib/constants';
import { DocumentTitleUpdateMessage, LocationChangeMessage } from '@pmm/shared';
import messenger from 'lib/messenger';
import { getLocationUrl } from './grafana.utils';
import { updateDocumentTitle } from 'lib/utils/document.utils';
import { useKioskMode } from 'hooks/utils/useKioskMode';

type Mode = 'light' | 'dark';
const readHtmlMode = (): Mode =>
  document.documentElement
    .getAttribute('data-md-color-scheme')
    ?.includes('dark')
    ? 'dark'
    : 'light';

export const GrafanaProvider: FC<PropsWithChildren> = ({ children }) => {
  const navigationType = useNavigationType();
  const location = useLocation();
  const src = location.pathname.replace(PMM_NEW_NAV_PATH, '');
  const isGrafanaPage = src.startsWith(GRAFANA_SUB_PATH);
  const [isLoaded, setIsloaded] = useState(false);
  const frameRef = useRef<HTMLIFrameElement>(null);
  const navigate = useNavigate();
  const kioskMode = useKioskMode();

  // Remember last theme we sent to avoid resending the same value.
  const lastSentThemeRef = useRef2<Mode>(readHtmlMode());

  useEffect(() => {
    if (isGrafanaPage) {
      setIsloaded(true);
    }
  }, [isGrafanaPage]);

  useEffect(() => {
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
    if (!isLoaded) return;

    messenger
      .setTargetWindow(frameRef.current?.contentWindow!, '#grafana-iframe')
      .register();

    // Send the current canonical theme to Grafana once messenger is ready.
    messenger.waitForMessage('MESSENGER_READY').then(() => {
      const mode = readHtmlMode(); // ✅ take from <html>, already synced with Grafana Prefs
      if (lastSentThemeRef.current !== mode) {
        lastSentThemeRef.current = mode;
        messenger.sendMessage({
          type: 'CHANGE_THEME',
          payload: { theme: mode },
        });
      }
    });

    messenger.addListener({
      type: 'LOCATION_CHANGE',
      onMessage: ({ payload: location }: LocationChangeMessage) => {
        if (!location || location.action === 'POP') return;
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

  // If outer theme changes (our hook updates <html> and posts messages), reflect it to Grafana.
  useEffect(() => {
    if (!isLoaded) return;
    const mode = readHtmlMode(); // ✅ canonical
    if (lastSentThemeRef.current !== mode) {
      lastSentThemeRef.current = mode;
      messenger.sendMessage({ type: 'CHANGE_THEME', payload: { theme: mode } });
    }
  }, [isLoaded, location]); // re-evaluate on navigation; inexpensive and safe

  return (
    <GrafanaContext.Provider
      value={{
        frameRef,
        isFrameLoaded: isLoaded,
        isOnGrafanaPage: isGrafanaPage,
        isFullScreen: kioskMode.active,
      }}
    >
      {children}
    </GrafanaContext.Provider>
  );
};

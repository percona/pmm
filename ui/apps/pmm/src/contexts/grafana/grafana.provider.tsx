import { FC, PropsWithChildren, useEffect, useRef, useState } from 'react';
import { useLocation, useNavigate, useNavigationType } from 'react-router';
import { GrafanaContext } from './grafana.context';
import {
  GRAFANA_SUB_PATH,
  PMM_NEW_NAV_GRAFANA_PATH,
  PMM_NEW_NAV_PATH,
} from 'lib/constants';
import type {
  ColorMode,
  LocationState,
  DocumentTitleUpdateMessage,
  LocationChangeMessage
} from '@pmm/shared';
import { updateDocumentTitle } from 'lib/utils/document.utils';
import { useKioskMode } from 'hooks/utils/useKioskMode';
import { useColorMode } from 'hooks/theme';
import { getLocationUrl } from './grafana.utils';
import messenger from 'lib/messenger';

/** Guard DOM usage. */
const isBrowser = () =>
  typeof window !== 'undefined' &&
  typeof window.addEventListener === 'function';

/**
 * GrafanaProvider wires three bridges:
 * 1) THEME: PMM (left) ↔ Grafana iframe (right)
 * 2) ROUTING: PMM (left) ↔ Grafana iframe (right)
 * 3) DOCUMENT TITLE: Grafana → PMM
 */
export const GrafanaProvider: FC<PropsWithChildren> = ({ children }) => {
  const navigationType = useNavigationType();
  const location = useLocation();
  const navigate = useNavigate();

  const src = location.pathname.replace(PMM_NEW_NAV_PATH, '');
  const isGrafanaPage = src.startsWith(GRAFANA_SUB_PATH);

  const [isLoaded, setIsLoaded] = useState(false);
  const frameRef = useRef<HTMLIFrameElement>(null);
  const kioskMode = useKioskMode();

  // Theme source
  const { colorMode, setFromGrafana } = useColorMode();

  useEffect(() => {
    if (isGrafanaPage) setIsLoaded(true);
  }, [isGrafanaPage]);

  // Register messenger, set iframe target, and add INCOMING listeners
  useEffect(() => {
    if (!isLoaded || !isBrowser()) return;

    const target = frameRef.current?.contentWindow;
    if (target) {
      messenger.setTargetWindow(target);
    }
    messenger.register();

    // -------- INCOMING FROM GRAFANA --------

    // Theme: apply without re-broadcast/persist (avoid ping-pong)
    messenger.addListener({
      type: 'GRAFANA_THEME_CHANGED',
      onMessage: (message: { payload?: { theme?: ColorMode } }) => {
        // No normalization here — setFromGrafana already normalizes inside the hook.
        if (!message.payload?.theme) return;
        setFromGrafana(message.payload.theme).catch((err: unknown) => {
          console.warn('[GrafanaProvider] setFromGrafana failed:', err);
        });
      },
    });

    // Location: navigate PMM when Grafana pushes/replace (skip POP/back)
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

    // Document title
    messenger.addListener({
      type: 'DOCUMENT_TITLE_CHANGE',
      onMessage: ({ payload }: DocumentTitleUpdateMessage) => {
        if (payload?.title) updateDocumentTitle(payload.title);
      },
    });

    // Cleanup once provider unmounts
    return () => {
      messenger.unregister();
    };
  }, [isLoaded, setFromGrafana, navigate]);

  // -------- OUTGOING TO GRAFANA --------

  // PMM -> Grafana: propagate PMM location (except if it came from Grafana)
  useEffect(() => {
    if (!isBrowser() || !isLoaded) return;

    const state = location.state as LocationState;
    if (!location.pathname.includes('/graph') || state?.fromGrafana) return;

    messenger.sendMessage({
      type: 'LOCATION_CHANGE',
      payload: {
        ...location,
        // Strip PMM wrapper prefix before sending to Grafana
        pathname: location.pathname.replace(PMM_NEW_NAV_GRAFANA_PATH, ''),
        action: navigationType,
      },
    });
  }, [location, navigationType, isLoaded]);

  // PMM -> Grafana: propagate theme when left-side theme changes
  useEffect(() => {
    if (!isLoaded || !isBrowser()) return;
    messenger.sendMessage({
      type: 'CHANGE_THEME',
      payload: { theme: colorMode }, // no extra normalization
    });
  }, [colorMode, isLoaded]);

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

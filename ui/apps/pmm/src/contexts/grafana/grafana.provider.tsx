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
  LocationChangeMessage,
} from '@pmm/shared';
import { updateDocumentTitle } from 'utils/document.utils';
import { useKioskMode } from 'hooks/utils/useKioskMode';
import { useAuth } from 'contexts/auth';
import { useColorMode } from 'hooks/theme';
import { getLocationUrl, isMigratedPage } from './grafana.utils';
import messenger from 'lib/messenger';
import { useSettings, useFrontendSettings } from 'hooks/api/useSettings';
import { useServiceTypes } from 'hooks/api/useServices';
import { useQueryClient } from '@tanstack/react-query';
import { USER_PREFERENCES_QUERY_KEY } from 'hooks/api/useUser';
import { isGrafanaLoginPath } from 'contexts/auth/auth.clientSession';
import { handleGrafanaUserLoggedOut } from 'contexts/auth/auth.grafanaLogout';

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
  const isGrafanaPageRef = useRef<boolean>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { isLoggedIn } = useAuth();

  const { refetch: refetchSettings } = useSettings({
    enabled: false,
  });
  const { data: frontendSettings, refetch: refetchFrontendSettings } =
    useFrontendSettings({ retry: false });
  const { refetch: refetchServiceTypes } = useServiceTypes({
    enabled: false,
  });

  const src = location.pathname.replace(PMM_NEW_NAV_PATH, '');
  const isGrafanaPage =
    src.startsWith(GRAFANA_SUB_PATH) && !isMigratedPage(src);
  isGrafanaPageRef.current = isGrafanaPage;

  const [isLoaded, setIsLoaded] = useState(false);
  const [grafanaDocumentTitle, setGrafanaDocumentTitle] = useState<string | null>(null);
  const frameRef = useRef<HTMLIFrameElement>(null);
  const kioskMode = useKioskMode();

  // Theme source
  const { colorMode, setFromGrafana } = useColorMode();

  useEffect(() => {
    const canLoadGrafanaIframe =
      isLoggedIn || Boolean(frontendSettings?.anonymousEnabled);
    setIsLoaded(isGrafanaPage && canLoadGrafanaIframe);
  }, [isGrafanaPage, isLoggedIn, frontendSettings?.anonymousEnabled]);

  useEffect(() => {
    if (!isGrafanaPage) {
      setGrafanaDocumentTitle(null);
    }
  }, [isGrafanaPage]);

  // Register messenger, set iframe target, and add INCOMING listeners
  useEffect(() => {
    if (!isLoaded || !isBrowser()) return;

    const target = frameRef.current?.contentWindow;
    if (target) {
      messenger.setTargetWindow(target, '#grafana-iframe');
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
          // eslint-disable-next-line no-console
          console.warn('[GrafanaProvider] setFromGrafana failed:', err);
        });
      },
    });

    // Location: navigate PMM when Grafana pushes/replace (skip POP/back)
    messenger.addListener({
      type: 'LOCATION_CHANGE',
      onMessage: ({ payload: location }: LocationChangeMessage) => {
        if (
          !location ||
          // dont navigate if we are not on grafana page
          !isGrafanaPageRef.current
        ) {
          return;
        }

        if (isGrafanaLoginPath(location.pathname)) {
          handleGrafanaUserLoggedOut(queryClient);
          return;
        }
        navigate(getLocationUrl(location), {
          state: { fromGrafana: true },
          replace: true,
        });
      },
    });

    // Document title (browser tab + ADRE chat context when on Grafana routes)
    messenger.addListener({
      type: 'DOCUMENT_TITLE_CHANGE',
      onMessage: ({ payload }: DocumentTitleUpdateMessage) => {
        if (!payload?.title) {
          return;
        }
        updateDocumentTitle(payload.title);
        if (typeof window !== 'undefined' && window.location.pathname.includes('/graph')) {
          setGrafanaDocumentTitle(payload.title);
        }
      },
    });

    messenger.addListener({
      type: 'SETTINGS_CHANGED',
      onMessage: () => refetchSettings(),
    });

    messenger.addListener({
      type: 'FRONTEND_SETTINGS_CHANGED',
      onMessage: () => refetchFrontendSettings(),
    });

    messenger.addListener({
      type: 'SERVICE_ADDED',
      onMessage: () => refetchServiceTypes(),
    });

    messenger.addListener({
      type: 'SERVICE_DELETED',
      onMessage: () => refetchServiceTypes(),
    });

    messenger.addListener({
      type: 'TIMEZONE_CHANGED',
      onMessage: () => {
        queryClient.invalidateQueries({ queryKey: USER_PREFERENCES_QUERY_KEY });
      },
    });

    // Cleanup once provider unmounts
    return () => {
      messenger.unregister();
    };

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isLoaded, setFromGrafana, navigate]);

  // -------- OUTGOING TO GRAFANA --------

  // PMM -> Grafana: propagate PMM location (except if it came from Grafana)
  useEffect(() => {
    if (!isBrowser() || !isLoaded) return;

    const isGrafanaPage = location.pathname.includes('/graph');
    const isSourceGrafana = (location.state as LocationState)?.fromGrafana;
    const isBackNavigation = navigationType === 'POP';

    if (!isGrafanaPage || (isSourceGrafana && !isBackNavigation)) {
      return;
    }

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
        grafanaDocumentTitle,
      }}
    >
      {children}
    </GrafanaContext.Provider>
  );
};

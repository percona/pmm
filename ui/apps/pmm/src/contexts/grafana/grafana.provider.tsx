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
import { useMessengerListener } from 'hooks/utils/useMessengerListener';
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

/**
 * GrafanaProvider wires three bridges:
 * 1) THEME: PMM (left) ↔ Grafana iframe (right)
 * 2) ROUTING: PMM (left) ↔ Grafana iframe (right)
 * 3) DOCUMENT TITLE: Grafana → PMM
 *
 * The messenger itself is registered once in `lib/messenger`; this provider only
 * owns subscriptions, each of which is torn down individually on unmount.
 */
export const GrafanaProvider: FC<PropsWithChildren> = ({ children }) => {
  const navigationType = useNavigationType();
  const location = useLocation();
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

  const [isLoaded, setIsLoaded] = useState(false);
  const frameRef = useRef<HTMLIFrameElement>(null);
  const kioskMode = useKioskMode();

  // Theme source
  const { colorMode, setFromGrafana } = useColorMode();

  useEffect(() => {
    const canLoadGrafanaIframe =
      isLoggedIn || Boolean(frontendSettings?.anonymousEnabled);
    setIsLoaded(isGrafanaPage && canLoadGrafanaIframe);
  }, [isGrafanaPage, isLoggedIn, frontendSettings?.anonymousEnabled]);

  // -------- INCOMING FROM GRAFANA --------

  // Theme: apply without re-broadcast/persist (avoid ping-pong)
  useMessengerListener<'GRAFANA_THEME_CHANGED', { theme?: ColorMode }>(
    'GRAFANA_THEME_CHANGED',
    (message) => {
      // No normalization here — setFromGrafana already normalizes inside the hook.
      if (!message.payload?.theme) return;
      setFromGrafana(message.payload.theme).catch((err: unknown) => {
        // eslint-disable-next-line no-console
        console.warn('[GrafanaProvider] setFromGrafana failed:', err);
      });
    }
  );

  // Location: navigate PMM when Grafana pushes/replace (skip POP/back)
  useMessengerListener(
    'LOCATION_CHANGE',
    ({ payload: location }: LocationChangeMessage) => {
      if (
        !location ||
        // dont navigate if we are not on grafana page
        !isGrafanaPage
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
    }
  );

  // Document title
  useMessengerListener(
    'DOCUMENT_TITLE_CHANGE',
    ({ payload }: DocumentTitleUpdateMessage) => {
      if (payload?.title) updateDocumentTitle(payload.title);
    }
  );

  useMessengerListener('SETTINGS_CHANGED', () => {
    refetchSettings();
  });

  useMessengerListener('FRONTEND_SETTINGS_CHANGED', () => {
    refetchFrontendSettings();
  });

  useMessengerListener('SERVICE_ADDED', () => {
    refetchServiceTypes();
  });

  useMessengerListener('SERVICE_DELETED', () => {
    refetchServiceTypes();
  });

  useMessengerListener('TIMEZONE_CHANGED', () => {
    queryClient.invalidateQueries({ queryKey: USER_PREFERENCES_QUERY_KEY });
  });

  // -------- OUTGOING TO GRAFANA --------
  // Not gated on the iframe being mounted: the messenger buffers state-sync
  // messages and replays them once Grafana announces itself.

  // PMM -> Grafana: propagate PMM location (except if it came from Grafana)
  useEffect(() => {
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
  }, [location, navigationType]);

  // PMM -> Grafana: propagate theme when left-side theme changes
  useEffect(() => {
    messenger.sendMessage({
      type: 'CHANGE_THEME',
      payload: { theme: colorMode }, // no extra normalization
    });
  }, [colorMode]);

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

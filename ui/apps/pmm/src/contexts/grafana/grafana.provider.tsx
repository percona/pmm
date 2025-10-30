import { FC, PropsWithChildren, useEffect, useRef, useState } from 'react';
import { useLocation, useNavigate, useNavigationType } from 'react-router';
import { GrafanaContext } from './grafana.context';
import {
  GRAFANA_SUB_PATH,
  PMM_NEW_NAV_GRAFANA_PATH,
  PMM_NEW_NAV_PATH,
} from 'lib/constants';
import { ColorMode, NavState } from '@pmm/shared';
import { getLocationUrl } from './grafana.utils';
import { updateDocumentTitle } from 'lib/utils/document.utils';
import { useKioskMode } from 'hooks/utils/useKioskMode';
import { useColorMode } from 'hooks/theme';
import { useSetTheme } from 'themes/setTheme';
import messenger from 'lib/messenger';

// ---- Minimal message shapes to avoid `any` ----
type ThemeChangedPayload = { theme?: ColorMode };
type LocationChangedPayload = {
  pathname?: string;
  search?: string;
  hash?: string;
  // We only guard for 'POP'
  action?: 'PUSH' | 'POP' | 'REPLACE' | string;
} & Record<string, unknown>;
type TitleChangedPayload = { title?: string };

type MessengerMessage<P> = {
  type: string;
  payload?: P;
};

type GrafanaLocationParam = Parameters<typeof getLocationUrl>[0];

const isBrowser = () =>
  typeof window !== 'undefined' &&
  typeof window.addEventListener === 'function';

export const GrafanaProvider: FC<PropsWithChildren> = ({ children }) => {
  const navigationType = useNavigationType();
  const location = useLocation();
  const src = location.pathname.replace(PMM_NEW_NAV_PATH, '');
  const isGrafanaPage = src.startsWith(GRAFANA_SUB_PATH);

  const [isLoaded, setIsLoaded] = useState(false);
  const frameRef = useRef<HTMLIFrameElement>(null);
  const navigate = useNavigate();
  const kioskMode = useKioskMode();

  // Left-side theme source of truth
  const { colorMode } = useColorMode();
  // Deterministic local apply that does not broadcast/persist (prevents ping-pong)
  const { setFromGrafana } = useSetTheme();

  // Avoid sending the same theme repeatedly
  const lastSentThemeRef = useRef<ColorMode>('light');

  useEffect(() => {
    if (isGrafanaPage) setIsLoaded(true);
  }, [isGrafanaPage]);

  // Register messenger for iframe and handle incoming messages
  useEffect(() => {
    if (!isLoaded || !isBrowser()) return;

    // setTargetWindow expects Window, so ensure contentWindow exists
    const target = frameRef.current?.contentWindow;
    if (target) {
      messenger.setTargetWindow(target);
    }
    messenger.register();

    // Grafana -> PMM: theme changed inside iframe
    messenger.addListener({
      type: 'GRAFANA_THEME_CHANGED',
      onMessage: (message: MessengerMessage<ThemeChangedPayload>) => {
        // Defensive read: normalize to 'light' | 'dark'
        const next: ColorMode =
          message.payload?.theme === 'dark' ? 'dark' : 'light';
        // Apply locally without re-broadcast/persist
        setFromGrafana(next).catch((err: unknown) => {
          console.warn('[GrafanaProvider] setFromGrafana failed:', err);
        });
      },
    });

    // Grafana -> PMM: route changes (except POP)
    messenger.addListener({
      type: 'LOCATION_CHANGE',
      onMessage: (message: {
        type: string;
        payload?: LocationChangedPayload;
      }) => {
        const loc = message.payload;
        if (!loc || loc.action === 'POP') return;

        // Adapt incoming payload to the exact type expected by getLocationUrl
        const adapted = loc as unknown as GrafanaLocationParam;

        navigate(getLocationUrl(adapted), {
          state: { fromGrafana: true },
          replace: true,
        });
      },
    });

    // Grafana -> PMM: document title
    messenger.addListener({
      type: 'DOCUMENT_TITLE_CHANGE',
      onMessage: (message: MessengerMessage<TitleChangedPayload>) => {
        const payload = message.payload;
        if (!payload?.title) return;
        updateDocumentTitle(payload.title);
      },
    });

    // Cleanup once provider unmounts
    return () => {
      messenger.unregister();
    };
  }, [isLoaded, navigate, setFromGrafana]);

  // PMM -> Grafana: propagate location (except when it came from Grafana)
  useEffect(() => {
    if (!isBrowser()) return;

    const state = location.state as NavState;
    if (!location.pathname.includes('/graph') || state?.fromGrafana) return;

    messenger.sendMessage({
      type: 'LOCATION_CHANGE',
      payload: {
        ...location,
        pathname: location.pathname.replace(PMM_NEW_NAV_GRAFANA_PATH, ''),
        action: navigationType,
      },
    });
  }, [location, navigationType]);

  // PMM -> Grafana: propagate theme when it changes on the left
  useEffect(() => {
    if (!isLoaded || !isBrowser()) return;

    const mode: ColorMode = colorMode === 'dark' ? 'dark' : 'light';
    if (lastSentThemeRef.current !== mode) {
      lastSentThemeRef.current = mode;
      messenger.sendMessage({
        type: 'CHANGE_THEME',
        payload: { theme: mode },
      });
    }
  }, [isLoaded, colorMode]);

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

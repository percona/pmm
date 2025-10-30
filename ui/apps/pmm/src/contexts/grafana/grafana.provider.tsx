import { FC, PropsWithChildren, useEffect, useRef, useState } from 'react';
import { useLocation, useNavigate, useNavigationType } from 'react-router';
import { GrafanaContext } from './grafana.context';
import {
  GRAFANA_SUB_PATH,
  PMM_NEW_NAV_GRAFANA_PATH,
  PMM_NEW_NAV_PATH,
} from 'lib/constants';
import { ColorMode, LocationState } from '@pmm/shared';
import { useKioskMode } from 'hooks/utils/useKioskMode';
import { useColorMode } from 'hooks/theme';
import { useSetTheme } from 'themes/setTheme';
import { getLocationUrl } from './grafana.utils';
import messenger from 'lib/messenger';

/** Guard DOM usage. */
const isBrowser = () =>
  typeof window !== 'undefined' &&
  typeof window.addEventListener === 'function';

/**
 * GrafanaProvider wires two bridges:
 * 1) THEME: PMM (left) ↔ Grafana iframe (right)
 * 2) ROUTING: PMM (left) ↔ Grafana iframe (right)
 *
 * Scope limited to the above; document title and other concerns are excluded.
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

  // Theme sources
  const { colorMode } = useColorMode();
  const { setFromGrafana } = useSetTheme();
  // const lastSentThemeRef = useRef<ColorMode>('light');

  useEffect(() => {
    if (isGrafanaPage) setIsLoaded(true);
  }, [isGrafanaPage]);

  // Register messenger, set iframe target, and add incoming listeners
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
        const next: ColorMode =
          message.payload?.theme === 'dark' ? 'dark' : 'light';
        setFromGrafana(next).catch((err: unknown) => {
          console.warn('[GrafanaProvider] setFromGrafana failed:', err);
        });
      },
    });

    // Location: navigate PMM when Grafana pushes/replace (skip POP/back)
    messenger.addListener({
      type: 'LOCATION_CHANGE',
      onMessage: (message: {
        payload?: {
          pathname?: string;
          search?: string;
          hash?: string;
          action?: string;
        };
      }) => {
        const loc = message.payload;
        if (!loc || loc.action === 'POP') return;

        // Pick only fields that getLocationUrl expects (no any/unknown)
        const adapted = {
          ...(typeof loc.pathname === 'string'
            ? { pathname: loc.pathname }
            : {}),
          ...(typeof loc.search === 'string' ? { search: loc.search } : {}),
          ...(typeof loc.hash === 'string' ? { hash: loc.hash } : {}),
        } as Parameters<typeof getLocationUrl>[0];

        navigate(getLocationUrl(adapted), {
          state: { fromGrafana: true },
          replace: true,
        });
      },
    });

    // Grafana -> PMM: document title
    messenger.addListener({
      type: 'DOCUMENT_TITLE_CHANGE',
      onMessage: ({ payload }: { payload?: { title?: string } }) => {
        if (payload?.title) document.title = payload.title;
      },
    });

    // Cleanup once provider unmounts
    return () => {
      messenger.unregister();
    };
  }, [isLoaded, navigate, setFromGrafana]);

  // -------- OUTGOING TO GRAFANA --------
  // PMM -> Grafana: propagate PMM location (except if it came from Grafana)
  useEffect(() => {
    if (!isBrowser()) return;

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
  }, [location, navigationType]);

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

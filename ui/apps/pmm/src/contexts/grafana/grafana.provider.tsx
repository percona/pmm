import { FC, PropsWithChildren, useEffect, useRef, useState } from 'react';
import { useLocation, useNavigate, useNavigationType } from 'react-router';
import { GrafanaContext } from './grafana.context';
import {
  GRAFANA_SUB_PATH,
  PMM_NEW_NAV_GRAFANA_PATH,
  PMM_NEW_NAV_PATH,
} from 'lib/constants';
import { DocumentTitleUpdateMessage, LocationChangeMessage } from '@pmm/shared';
import { getLocationUrl } from './grafana.utils';
import { updateDocumentTitle } from 'lib/utils/document.utils';
import { useKioskMode } from 'hooks/utils/useKioskMode';
import { useColorMode } from 'hooks/theme';
import { useSetTheme } from 'themes/setTheme';

type Mode = 'light' | 'dark';

const isBrowser = (): boolean =>
  typeof window !== 'undefined' && typeof window.addEventListener === 'function';

/** Reads canonical mode from <html> attributes set by our theme hook. */
const readHtmlMode = (): Mode => {
  if (!isBrowser()) return 'light';
  return document.documentElement
    .getAttribute('data-md-color-scheme')
    ?.includes('dark')
    ? 'dark'
    : 'light';
};

/** Normalizes any incoming value to 'light' | 'dark'. */
const normalizeMode = (v: unknown): Mode =>
  typeof v === 'string' && v.toLowerCase() === 'dark'
    ? 'dark'
    : v === true
      ? 'dark'
      : 'light';

/** Resolve optional Grafana origin provided via env (e.g. https://pmm.example.com). */
const resolveGrafanaOrigin = (): string | undefined => {
  // Import meta may be undefined in tests; guard access.
  const raw = (import.meta as ImportMeta | undefined)?.env?.VITE_GRAFANA_ORIGIN as
    | string
    | undefined;
  if (!raw) return undefined;
  try {
    return new URL(raw).origin;
  } catch {
    return undefined;
  }
};

/** Build a trust predicate for postMessage origins. */
const makeIsTrustedOrigin = () => {
  // In dev, accept any origin to support split hosts/ports (vite + docker)
  if ((import.meta as ImportMeta | undefined)?.env?.DEV) return () => true;

  if (!isBrowser()) return () => false;
  const set = new Set<string>([window.location.origin]);
  const grafanaOrigin = resolveGrafanaOrigin();
  if (grafanaOrigin) set.add(grafanaOrigin);
  return (origin: string) => set.has(origin);
};

export const GrafanaProvider: FC<PropsWithChildren> = ({ children }) => {
  const navigationType = useNavigationType();
  const location = useLocation();
  const src = location.pathname.replace(PMM_NEW_NAV_PATH, '');
  const isGrafanaPage = src.startsWith(GRAFANA_SUB_PATH);

  const [isLoaded, setIsLoaded] = useState(false);
  const frameRef = useRef<HTMLIFrameElement>(null);
  const navigate = useNavigate();
  const kioskMode = useKioskMode();

  // Ensure our theme context is mounted (also mounts the global theme sync hook elsewhere)
  useColorMode();

  const { setFromGrafana } = useSetTheme();

  // Remember last theme we sent to avoid resending the same value.
  // Do not read document during initial render (tests/SSR friendly).
  const lastSentThemeRef = useRef<Mode>('light');

  // Keep messenger instance lazily loaded and scoped to browser only.
  const messengerRef = useRef<any | null>(null);

  // Trusted-origin predicate for postMessage validation.
  const isTrustedOriginRef = useRef<(o: string) => boolean>(() => true);
  useEffect(() => {
    isTrustedOriginRef.current = makeIsTrustedOrigin();
  }, []);

  // Mark iframe area as loaded when we hit /graph/*
  useEffect(() => {
    if (isGrafanaPage) setIsLoaded(true);
  }, [isGrafanaPage]);

  // Lazily import and register messenger when iframe area is ready (browser only).
  useEffect(() => {
    if (!isLoaded || !isBrowser()) return;

    let mounted = true;
    (async () => {
      try {
        const mod = await import('lib/messenger');
        if (!mounted) return;

        const messenger = mod.default;
        messengerRef.current = messenger;

        messenger.setTargetWindow(frameRef.current?.contentWindow!, '#grafana-iframe').register();

        // Initialize lastSentThemeRef from DOM now that we are in browser.
        lastSentThemeRef.current = readHtmlMode();

        // Send the current canonical theme to Grafana once messenger is ready.
        messenger.waitForMessage?.('MESSENGER_READY').then(() => {
          const mode = readHtmlMode();
          if (lastSentThemeRef.current !== mode) {
            lastSentThemeRef.current = mode;
            messenger.sendMessage?.({
              type: 'CHANGE_THEME',
              payload: { theme: mode },
            });
          }
        });

        // Mirror Grafana → PMM route changes (except POP)
        messenger.addListener?.({
          type: 'LOCATION_CHANGE',
          onMessage: ({ payload: loc }: LocationChangeMessage) => {
            if (!loc || (loc as any).action === 'POP') return;
            navigate(getLocationUrl(loc), {
              state: { fromGrafana: true },
              replace: true,
            });
          },
        });

        // Mirror Grafana document title
        messenger.addListener?.({
          type: 'DOCUMENT_TITLE_CHANGE',
          onMessage: ({ payload }: DocumentTitleUpdateMessage) => {
            if (!payload) return;
            updateDocumentTitle(payload.title);
          },
        });
      } catch (err) {
        // eslint-disable-next-line no-console
        console.warn('[GrafanaProvider] lazy messenger setup failed:', err);
      }
    })();

    return () => {
      mounted = false;
      try {
        messengerRef.current?.unregister?.();
      } catch {
        // no-op
      }
      messengerRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isLoaded]);

  // Propagate location changes to Grafana (except POP from Grafana itself)
  useEffect(() => {
    if (!isBrowser()) return;
    if (
      !location.pathname.includes('/graph') ||
      (location.state as any)?.fromGrafana
    ) {
      return;
    }
    const messenger = messengerRef.current;
    if (!messenger) return;

    messenger.sendMessage?.({
      type: 'LOCATION_CHANGE',
      payload: {
        ...location,
        pathname: location.pathname.replace(PMM_NEW_NAV_GRAFANA_PATH, ''),
        action: navigationType,
      },
    });
  }, [location, navigationType]);

  // If outer theme changes (our hook updates <html>), reflect it to Grafana quickly
  useEffect(() => {
    if (!isLoaded || !isBrowser()) return;
    const mode = readHtmlMode(); // canonical
    if (lastSentThemeRef.current !== mode) {
      lastSentThemeRef.current = mode;
      messengerRef.current?.sendMessage?.({
        type: 'CHANGE_THEME',
        payload: { theme: mode },
      });
    }
  }, [isLoaded, location]); // re-evaluate on navigation; inexpensive and safe

  // Hard guarantee: listen for grafana.theme.changed on /graph/* pages and apply locally (no persist/broadcast).
  useEffect(() => {
    if (!isLoaded || !isBrowser()) return;

    const onMsg = (
      e: MessageEvent<{
        type?: string;
        payload?: { mode?: string; payloadMode?: string; isDark?: boolean };
      }>
    ) => {
      // Security: ignore unexpected origins in production
      if (!isTrustedOriginRef.current(e.origin)) return;

      if (!e?.data || e.data.type !== 'grafana.theme.changed') return;
      const p = e.data.payload ?? {};
      const raw = p.mode ?? p.payloadMode ?? (p.isDark ? 'dark' : 'light');
      const desired = normalizeMode(raw);

      // Apply locally only to avoid ping-pong; persistence is handled by left action.
      setFromGrafana(desired).catch((err) =>
        // eslint-disable-next-line no-console
        console.warn('[GrafanaProvider] setFromGrafana failed:', err)
      );
    };

    window.addEventListener('message', onMsg);
    return () => window.removeEventListener('message', onMsg);
  }, [isLoaded, setFromGrafana]);

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

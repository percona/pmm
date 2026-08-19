import {
  FC,
  PropsWithChildren,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { useServerVersion } from 'hooks/api/useServerVersion';
import { reloadPage } from 'utils/dom.utils';
import { VersionContext } from './version.context';
import {
  canAutoReload,
  getServerBuildId,
  hasBuildChanged,
  recordAutoReload,
} from './version.utils';

/**
 * Keeps the page in sync with the server build. An external upgrade (Docker, Podman,
 * Helm) swaps the backend under an open tab, which then keeps running the assets of
 * the previous version and renders a mix of the two, so the page has to be reloaded.
 *
 * A hidden tab is reloaded straight away, which covers the case this exists for: a
 * tab left open in the background while the server is upgraded. A visible tab is
 * left alone so the reload cannot discard work in progress, including unsaved edits
 * inside the Grafana iframe; consumers prompt for that case instead.
 */
export const VersionProvider: FC<PropsWithChildren> = ({ children }) => {
  const { data } = useServerVersion();
  const [isOutdated, setIsOutdated] = useState(false);
  const baseline = useRef('');
  const reloading = useRef(false);
  const buildId = getServerBuildId(data);

  // Leaving the document fires `visibilitychange` on the way out, so a reload
  // already under way would otherwise look like a tab being backgrounded and
  // trigger a second one.
  const reload = useCallback(() => {
    if (reloading.current) {
      return;
    }

    reloading.current = true;
    reloadPage();
  }, []);

  // Reloading on our own initiative is rationed; doing it because the user asked
  // is not, so only this path spends the allowance.
  const autoReload = useCallback(() => {
    if (reloading.current || !canAutoReload(Date.now())) {
      return;
    }

    recordAutoReload(Date.now());
    reload();
  }, [reload]);

  useEffect(() => {
    if (!buildId) {
      return;
    }

    if (!baseline.current) {
      baseline.current = buildId;
      return;
    }

    if (hasBuildChanged(baseline.current, buildId)) {
      setIsOutdated(true);
    }
  }, [buildId]);

  useEffect(() => {
    if (!isOutdated) {
      return;
    }

    const reloadWhenHidden = () => {
      if (document.visibilityState === 'hidden') {
        autoReload();
      }
    };

    reloadWhenHidden();
    document.addEventListener('visibilitychange', reloadWhenHidden);

    return () => {
      document.removeEventListener('visibilitychange', reloadWhenHidden);
    };
  }, [isOutdated, autoReload]);

  // Lazily imported chunks are named after their content, so an upgrade leaves the
  // ones this page knows about missing from the server. That surfaces before the
  // next poll does and has no recovery other than loading the new assets.
  useEffect(() => {
    window.addEventListener('vite:preloadError', autoReload);

    return () => {
      window.removeEventListener('vite:preloadError', autoReload);
    };
  }, [autoReload]);

  const value = useMemo(
    () => ({
      isOutdated,
      serverVersion: data?.version ?? '',
      reload,
    }),
    [isOutdated, data?.version, reload]
  );

  return (
    <VersionContext.Provider value={value}>{children}</VersionContext.Provider>
  );
};

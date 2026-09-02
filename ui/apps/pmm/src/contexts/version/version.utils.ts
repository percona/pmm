import { GetServerVersionResponse } from 'types/server.types';

const AUTO_RELOAD_STORAGE_KEY = 'pmm.lastAutoReloadAt';

/**
 * How long to wait before allowing a second automatic reload. Without it, a tab
 * served in turn by an upgraded and a not yet upgraded HA node would see the build
 * change on every poll and reload forever.
 */
export const AUTO_RELOAD_COOLDOWN_MS = 10 * 60 * 1000;

/**
 * Identity of the build the server runs. `managed.fullVersion` is the git commit
 * injected through ldflags, so it also changes when an image is rebuilt under an
 * unchanged version, which is what release candidates and feature builds do. It is
 * empty for binaries built without ldflags, hence the fallbacks; an empty id leaves
 * the watcher inert rather than reloading on every poll.
 */
export const getServerBuildId = (data?: GetServerVersionResponse) =>
  data?.managed?.fullVersion ||
  data?.server?.fullVersion ||
  data?.version ||
  '';

export const hasBuildChanged = (baseline: string, current: string) =>
  !!baseline && !!current && baseline !== current;

const readLastAutoReloadAt = (): number | null => {
  try {
    const stored = Number(sessionStorage.getItem(AUTO_RELOAD_STORAGE_KEY));
    return Number.isFinite(stored) && stored > 0 ? stored : null;
  } catch {
    return null;
  }
};

export const canAutoReload = (now: number) => {
  const lastAt = readLastAutoReloadAt();
  return lastAt === null || now - lastAt >= AUTO_RELOAD_COOLDOWN_MS;
};

/**
 * Notes a reload against the allowance. Returns false when storage refused it,
 * which browsers configured to block site data do: the record is what makes the
 * allowance enforceable, so callers must treat a refusal as no allowance at all.
 */
export const recordAutoReload = (now: number) => {
  try {
    sessionStorage.setItem(AUTO_RELOAD_STORAGE_KEY, String(now));
    return true;
  } catch {
    return false;
  }
};

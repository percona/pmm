const STORAGE_KEY = 'pmm-native-qan-enabled';

/** Client-side snapshot of server flags for non-React code (e.g. ADRE frontend tools). */
let nativeQanEnabledSnapshot = readStoredFlag();

function readStoredFlag(): boolean {
  try {
    return sessionStorage.getItem(STORAGE_KEY) === 'true';
  } catch {
    return false;
  }
}

export function setNativeQanEnabledSnapshot(enabled: boolean): void {
  nativeQanEnabledSnapshot = enabled;
  try {
    sessionStorage.setItem(STORAGE_KEY, enabled ? 'true' : 'false');
  } catch {
    /* ignore */
  }
}

export function getNativeQanEnabledSnapshot(): boolean {
  return nativeQanEnabledSnapshot;
}

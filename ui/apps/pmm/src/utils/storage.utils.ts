/**
 * Web Storage access that cannot throw. Under Safari private browsing and any blocked-site-data
 * policy the accessors raise on property access, so the reference has to sit inside the try. The
 * app mounts no error boundary, so an unguarded throw on the boot path blanks the page; a failure
 * here just means "nothing stored".
 */
const safeStorage = (getStorage: () => Storage) => ({
  getItem(key: string): string | null {
    try {
      return getStorage().getItem(key);
    } catch {
      return null;
    }
  },

  setItem(key: string, value: string) {
    try {
      getStorage().setItem(key, value);
    } catch {
      // Unavailable: the value is lost, the caller carries on.
    }
  },

  removeItem(key: string) {
    try {
      getStorage().removeItem(key);
    } catch {
      // Nothing to clear if storage is unreachable.
    }
  },
});

export const safeSessionStorage = safeStorage(() => sessionStorage);
export const safeLocalStorage = safeStorage(() => localStorage);

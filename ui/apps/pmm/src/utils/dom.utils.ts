/**
 * Returns a promise that resolves when an element is available in DOM
 * @param selector query selector of the element
 * @returns resolves when element is available in DOM
 */
export const waitForVisible = (selector: string, timeout = 5000) =>
  new Promise<boolean>((resolve, reject) => {
    if (document.querySelector(selector)) {
      return resolve(true);
    }

    const observer = new MutationObserver(() => {
      if (typeof document === 'undefined') {
        observer.disconnect();
        return;
      }

      if (document.querySelector(selector)) {
        clearTimeout(timeoutId);
        resolve(true);
        observer.disconnect();
      }
    });

    const timeoutId = setTimeout(() => {
      observer.disconnect();
      reject();
    }, timeout);

    observer.observe(document.body, {
      childList: true,
      subtree: true,
    });
  });

const PMM_UPGRADE_QUERY_PARAM = 'pmmUpgrade';

/**
 * Reload the page after a PMM Server upgrade, bypassing cached HTML.
 */
export const hardReloadPage = (upgradeVersion: string) => {
  const url = new URL(window.location.href);
  url.searchParams.set(PMM_UPGRADE_QUERY_PARAM, upgradeVersion);
  window.location.replace(url.toString());
};

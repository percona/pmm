/**
 * Returns a promise that resolves when an element is available in DOM
 * @param selector query selector of the element
 * @returns resolves when element is available in DOM
 */
export const waitForVisible = (selector: string, timeout = 5000) =>
  new Promise<boolean>((resolve, reject) => {
    const timeoutId = setTimeout(() => reject(), timeout);

    if (document.querySelector(selector)) {
      clearTimeout(timeoutId);
      return resolve(true);
    }

    const observer = new MutationObserver(() => {
      if (document.querySelector(selector)) {
        clearTimeout(timeoutId);
        resolve(true);
        observer.disconnect();
      }
    });

    observer.observe(document.body, {
      childList: true,
      subtree: true,
    });
  });

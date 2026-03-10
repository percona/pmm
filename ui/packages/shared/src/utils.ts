/**
 * True when the page is loaded by the Grafana image renderer (or similar headless/automated context).
 * Unless in development or production environment, which might happen when running tests (e.g. Playwright).
 */
export const isHeadlessBrowser = (): boolean => {
  if (navigator === undefined || window === undefined) {
    return false;
  }

  if (navigator.webdriver) {
    return false;
  }
  // Common headless/automated browser signals
  if (/HeadlessChrome|Headless/i.test(navigator.userAgent)) {
    return true;
  }

  return false;
};

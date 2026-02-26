/**
 * True when the page is loaded by the Grafana image renderer (or similar headless/automated context).
 * Unless in development or production environment, which might happen when running tests (e.g. Playwright).
 */
export const isHeadlessBrowser = (): boolean => {
  if (typeof navigator === 'undefined' || typeof window === 'undefined') {
    return false;
  }
  // Common headless/automated browser signals
  if (
    /HeadlessChrome|Headless/i.test(navigator.userAgent) &&
    ['development', 'production'].includes(process.env.NODE_ENV || '')
  ) {
    return true;
  }

  return false;
};

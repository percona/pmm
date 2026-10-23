// Grafana stars and unstars a dashboard through the user stars API, by uid or
// by the older numeric id, or with the starsFromAPIServer feature through the
// collections API.
const STAR_ENDPOINTS = [
  /\/api\/user\/stars\/dashboard\/(uid\/)?[^/]+$/,
  /\/apis\/collections\.grafana\.app\/[^/]+\/namespaces\/[^/]+\/stars\/[^/]+\/update\//,
];
const STAR_METHODS = ['POST', 'PUT', 'DELETE'];

const describeRequest = (input: RequestInfo | URL, init?: RequestInit) => {
  const isAddress = typeof input === 'string' || input instanceof URL;
  const url = isAddress ? String(input) : input.url;
  const method = init?.method ?? (isAddress ? 'GET' : input.method);

  return {
    pathname: new URL(url, window.location.origin).pathname,
    method: method.toUpperCase(),
  };
};

/**
 * Calls `onChange` once Grafana has starred or unstarred a dashboard, wherever
 * in Grafana that happened, so PMM UI can update the sidebar's Starred section.
 * Grafana fires no event for it, so this watches the request that saves it.
 */
export const watchDashboardStars = (onChange: () => void) => {
  const fetchFn = window.fetch.bind(window);

  window.fetch = async (input, init) => {
    const response = await fetchFn(input, init);
    const { pathname, method } = describeRequest(input, init);

    if (
      response.ok &&
      STAR_METHODS.includes(method) &&
      STAR_ENDPOINTS.some((endpoint) => endpoint.test(pathname))
    ) {
      onChange();
    }

    return response;
  };
};

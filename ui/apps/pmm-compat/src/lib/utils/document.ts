export const documentTitleObserver = {
  listen: (callback: (title: string) => void) =>
    new MutationObserver(() => callback(document.title)).observe(document.querySelector('title')!, {
      subtree: true,
      characterData: true,
      childList: true,
    }),
};

export const updateBodyClassByLocation = (location: Location) => {
  const previous = Array.from(document.body.classList).find((className) => className.startsWith('grafana-compat-page'));
  if (previous) {
    document.body.classList.remove(previous);
  }
  const sanitizedPathname = location.pathname
    .replace(/[^a-zA-Z0-9/-]/g, '')
    .replace('/graph', '')
    .replaceAll('/', '-');
  const className = `grafana-compat-page${sanitizedPathname}`;
  document.body.classList.add(className);
};

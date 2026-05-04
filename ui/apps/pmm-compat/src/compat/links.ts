// Open external links in a new tab
export const handleExternalLinks = () =>
  window.addEventListener(
    'click',
    (e) => {
      const a = (e.target as HTMLElement)?.closest('a');

      if (!a) {
        return;
      }

      const url = a.getAttribute('href');

      if (!url) {
        return;
      }

      const urlObj = new URL(url, window.location.href);
      const isExternal = urlObj.origin !== window.location.origin;

      if (isExternal) {
        e.preventDefault();
        window.open(url, '_blank', 'noopener,noreferrer');
      }
    },
    {
      capture: true,
    }
  );

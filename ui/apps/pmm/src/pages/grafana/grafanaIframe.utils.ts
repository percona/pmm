export const getIframePathname = (
  iframe: HTMLIFrameElement | null | undefined
): string | null => {
  try {
    return iframe?.contentWindow?.location.pathname ?? null;
  } catch {
    return null;
  }
};

/** If the Grafana iframe navigated to the PMM shell, reload it with the Grafana URL. */
export const redirectIframeFromPmmShell = (
  iframe: HTMLIFrameElement,
  grafanaSrc: string
): boolean => {
  try {
    const pathname = iframe.contentWindow?.location.pathname;
    if (!pathname?.startsWith('/pmm-ui')) {
      return false;
    }
    iframe.src = grafanaSrc;
    return true;
  } catch {
    return false;
  }
};

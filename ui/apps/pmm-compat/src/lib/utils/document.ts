export const documentTitleObserver = {
  listen: (callback: (title: string) => void) =>
    new MutationObserver(() => callback(document.title)).observe(document.querySelector('title')!, {
      subtree: true,
      characterData: true,
      childList: true,
    }),
};

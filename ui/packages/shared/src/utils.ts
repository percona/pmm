export const isRenderingServer = (): boolean => {
  const params = new URLSearchParams(window.location.search);

  return params.get('render') === '1';
};

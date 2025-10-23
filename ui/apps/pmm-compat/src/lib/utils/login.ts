export const isFirstLogin = () => {
  const userId = window.grafanaBootData?.user?.id;
  return localStorage.getItem(`pmm-ui.first-login.user-${userId}`) !== 'false';
};

export const updateIsFirstLogin = () => {
  const userId = window.grafanaBootData?.user?.id;
  localStorage.setItem(`pmm-ui.first-login.user-${userId}`, 'false');
};

export const LOCATORS = {
  toolbar: 'header > div:first-child > div:nth-child(2)',
  menuToggle: 'header #mega-menu-toggle',
  helpButton: 'header button[aria-label="Help"]',
  searchButton: 'header button[aria-label="Search or jump to..."]',
  profileButton: 'header button[aria-label="Profile"]',
  commandPaletteTrigger: 'header div[data-testid="data-testid Command palette trigger"]',
};

export const GRAFANA_SUB_PATH = '/graph';
export const GRAFANA_LOGIN_PATH = '/graph/login';
export const GRAFANA_DOCKED_MENU_OPEN_LOCAL_STORAGE_KEY = 'grafana.navigation.open';
export const PMM_UI_PATH = `/pmm-ui/next${GRAFANA_SUB_PATH}`;

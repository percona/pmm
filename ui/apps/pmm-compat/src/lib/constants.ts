export const PAGE_LOCATORS = {
  qan: 'body.grafana-compat-page-d-pmm-qan-pmm-query-analytics',
};

export const LOCATORS = {
  toolbar: 'header > div:first-child > div:nth-child(2)',
  menuToggle: 'header #mega-menu-toggle',
  helpButton: 'header button[aria-label="Help"]',
  searchButton: 'header button[aria-label="Search or jump to..."]',
  profileButton: 'header button[aria-label="Profile"]',
  commandPaletteTrigger: 'header div[data-testid="data-testid Command palette trigger"]',
  toolbarSignIn: 'header > div:first-child > div:nth-child(2) > a[target="_self"]',
  qanPageHeader: `${PAGE_LOCATORS.qan} header`,
  qanPageHeaderNextDiv: `${PAGE_LOCATORS.qan} header+div`,
  qanPageCanvasWrapper: `${PAGE_LOCATORS.qan} [class*="canvas-wrapper"] > div`,
};

export const GRAFANA_SUB_PATH = '/graph';
export const GRAFANA_LOGIN_PATH = '/graph/login';
export const GRAFANA_DOCKED_MENU_OPEN_LOCAL_STORAGE_KEY = 'grafana.navigation.open';
export const PMM_UI_PATH = '/pmm-ui/next';
export const PMM_UI_GRAFANA_PATH = `${PMM_UI_PATH}${GRAFANA_SUB_PATH}`;
export const PMM_UI_HELP_PATH = `${PMM_UI_PATH}/help`;

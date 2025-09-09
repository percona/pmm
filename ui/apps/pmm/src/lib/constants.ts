import { AdvisorFamily, AdvisorInterval } from 'types/advisors.types';
import { ServiceType } from 'types/services.types';

export const PMM_TITLE = 'Percona Monitoring and Management';
export const PMM_NEW_NAV_PATH = '/next';
export const GRAFANA_SUB_PATH = '/graph';
export const PMM_BASE_PATH = `/pmm-ui${PMM_NEW_NAV_PATH}`;
export const PMM_NEW_NAV_GRAFANA_PATH = `${PMM_NEW_NAV_PATH}${GRAFANA_SUB_PATH}`;
export const PMM_HOME_URL = `${GRAFANA_SUB_PATH}/d/pmm-home`;
export const PMM_LOGIN_URL = `${GRAFANA_SUB_PATH}/login`;
export const PMM_SETTINGS_URL = `${GRAFANA_SUB_PATH}/settings/advanced-settings`;
export const PMM_SUPPORT_URL = 'https://per.co.na/pmm_documentation';
export const PMM_DOCS_UPDATES_URL = 'https://per.co.na/pmm-upgrade';
export const PMM_DOCS_UPDATE_CLIENT_URL = 'https://per.co.na/pmm-upgrade-agent';

export const INTERVALS_MS = {
  // 5 mins
  SERVICE_TYPES: 300000,
};

export const ADVISOR_FAMILY: Record<AdvisorFamily, string> = {
  [AdvisorFamily.mysql]: 'MySQL',
  [AdvisorFamily.postgresql]: 'PostgreSQL',
  [AdvisorFamily.mongodb]: 'MongoDB',
  [AdvisorFamily.unspecified]: 'Unspecified',
};

export const ADVISOR_INTERVAL: Record<AdvisorInterval, string> = {
  [AdvisorInterval.standard]: 'Standard',
  [AdvisorInterval.rare]: 'Rare',
  [AdvisorInterval.frequent]: 'Frequent',
  [AdvisorInterval.unspecified]: 'Unspecified',
};

export const ALL_SERVICE_TYPES = [
  ServiceType.external,
  ServiceType.haproxy,
  ServiceType.mongodb,
  ServiceType.mysql,
  ServiceType.posgresql,
  ServiceType.proxysql,
];

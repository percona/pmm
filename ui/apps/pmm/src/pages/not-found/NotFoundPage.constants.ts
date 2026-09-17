import { PMM_NEW_NAV_GRAFANA_PATH, PMM_NEW_NAV_PATH } from 'lib/constants';
import { Messages } from './NotFoundPage.messages';

export interface QuickLink {
  id: string;
  label: string;
  to: string;
}

export const QUICK_LINKS: QuickLink[] = [
  {
    id: 'inventory',
    label: Messages.links.inventory,
    to: `${PMM_NEW_NAV_GRAFANA_PATH}/inventory`,
  },
  {
    id: 'settings',
    label: Messages.links.settings,
    to: `${PMM_NEW_NAV_PATH}/settings`,
  },
  {
    id: 'help',
    label: Messages.links.help,
    to: `${PMM_NEW_NAV_PATH}/help`,
  },
];

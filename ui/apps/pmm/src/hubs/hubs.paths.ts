import { MONGODB_HUB_PATH } from 'lib/constants';
import type { NavItem } from 'types/navigation.types';

// Constants only, so the sidebar can point at hubs without loading their code.

/** Hub page path by the id of the sidebar entry that opens it. */
export const HUB_PATHS: Readonly<Record<string, string>> = {
  mongo: MONGODB_HUB_PATH,
};

/** Points a technology entry at its hub, keeping it highlighted on the dashboards it matched before. */
export const withHubPage = (item: NavItem): NavItem => {
  const hub = HUB_PATHS[item.id];

  return hub
    ? { ...item, url: hub, matches: [hub, ...(item.matches ?? [])] }
    : item;
};

import type { ComponentType } from 'react';
import type { SvgIconProps } from '@mui/material/SvgIcon';
import AccountTreeIcon from '@mui/icons-material/AccountTree';
import ArchiveIcon from '@mui/icons-material/Archive';
import AssignmentIcon from '@mui/icons-material/Assignment';
import BarChartIcon from '@mui/icons-material/BarChart';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CodeIcon from '@mui/icons-material/Code';
import DescriptionIcon from '@mui/icons-material/Description';
import ExtensionIcon from '@mui/icons-material/Extension';
import ScienceIcon from '@mui/icons-material/Science';
import SupportAgentIcon from '@mui/icons-material/SupportAgent';
import TableChartIcon from '@mui/icons-material/TableChart';
import TroubleshootIcon from '@mui/icons-material/Troubleshoot';
import { MongoIcon, MySqlIcon, PostgreSqlIcon } from '@percona/peak-ui';
import type { EnabledApp, ExtensionsComponents } from '@pmm-extensions/api';
import { EXTENSIONS_PATH } from 'lib/constants';

type NavIconKey = ExtensionsComponents['schemas']['nav_icons__NavIcon'];

export type ExtensionsAppIcon = ComponentType<SvgIconProps>;

/**
 * A PMM Extensions app as PMM's navigation and router see it. PMM owns this
 * shape, so another source for the list changes the mapping and nothing else.
 */
export interface ExtensionsApp {
  key: string;
  /** Path segment under `/extensions`, unique among the listed apps. */
  slug: string;
  label: string;
  icon: ExtensionsAppIcon;
  /** Sort position in the Apps section; `null` sorts last. */
  order: number | null;
  enabled: boolean;
  /** The app ships its own page, which the generic route cannot render. */
  customUi: boolean;
  /** Labels of the disabled apps that keep this one disabled. */
  blockedBy: string[];
}

/** A `Record` over the side-car's icon keys, so a new key fails type-checking until mapped. */
const ICON_BY_KEY: Record<NavIconKey, ExtensionsAppIcon> = {
  assignment: AssignmentIcon,
  code: CodeIcon,
  'support-agent': SupportAgentIcon,
  description: DescriptionIcon,
  troubleshoot: TroubleshootIcon,
  'table-chart': TableChartIcon,
  'check-circle': CheckCircleIcon,
  mysql: MySqlIcon,
  mongo: MongoIcon,
  postgresql: PostgreSqlIcon,
  archive: ArchiveIcon,
  science: ScienceIcon,
  'bar-chart': BarChartIcon,
  'account-tree': AccountTreeIcon,
};

export const DEFAULT_EXTENSIONS_APP_ICON: ExtensionsAppIcon = ExtensionIcon;

/** `mysql_backups` → `mysql-backups`, which keeps the URLs of the apps routed before the list. */
export const toExtensionsAppSlug = (appKey: string) =>
  appKey.replace(/_/g, '-');

export const extensionsAppPath = (slug: string) => `${EXTENSIONS_PATH}/${slug}`;

const byNavOrder = (a: ExtensionsApp, b: ExtensionsApp) => {
  if (a.order !== b.order) {
    if (a.order === null) {
      return 1;
    }
    if (b.order === null) {
      return -1;
    }
    return a.order - b.order;
  }
  return a.label.localeCompare(b.label);
};

/**
 * The apps that can have a place in the Apps section, in nav order. Child apps
 * (`mysql_backups/restore`) are reached through their parent, and apps without
 * `sidebar` serve an API only, so neither is kept. Disabled apps are, so the
 * route can tell a disabled app from an unknown one.
 */
export const toExtensionsApps = (apps: EnabledApp[]): ExtensionsApp[] => {
  const labels = new Map(apps.map((app) => [app.app_key, app.display_name]));

  return apps
    .filter((app) => app.sidebar && !app.app_key.includes('/'))
    .map((app) => ({
      key: app.app_key,
      slug: toExtensionsAppSlug(app.app_key),
      label: app.display_name,
      icon: app.nav_icon
        ? (ICON_BY_KEY[app.nav_icon] ?? DEFAULT_EXTENSIONS_APP_ICON)
        : DEFAULT_EXTENSIONS_APP_ICON,
      order: app.nav_order,
      enabled: app.enabled,
      customUi: app.custom_ui,
      blockedBy: (app.blocking_dependencies ?? []).map(
        (key) => labels.get(key) ?? key
      ),
    }))
    .sort(byNavOrder);
};

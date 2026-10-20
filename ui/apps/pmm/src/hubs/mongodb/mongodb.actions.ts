import AddLinkOutlined from '@mui/icons-material/AddLinkOutlined';
import ArrowOutwardOutlined from '@mui/icons-material/ArrowOutwardOutlined';
import DownloadOutlined from '@mui/icons-material/DownloadOutlined';
import NotificationsOutlined from '@mui/icons-material/NotificationsOutlined';
import SettingsBackupRestoreOutlined from '@mui/icons-material/SettingsBackupRestoreOutlined';
import { Graph4Icon, NetworkIntelligenceIcon } from '@percona/peak-ui';
import { ALERT_TEMPLATES_PATH } from 'hubs/template/hub.doors';
import type {
  HubCapabilities,
  HubRole,
  HubViewModel,
} from 'hubs/template/hub.types';
import { MONGO_PATHS } from './mongodb.doors';
import { Messages } from './mongodb.messages';

const M = Messages.actions;

/** The header's menus: how an admin brings MongoDB in, the estate-wide admin pages, and the docs for everyone. */
export const deriveActions = (
  capabilities: Pick<HubCapabilities, 'addService' | 'install' | 'setUpBackup'>,
  role: HubRole
): HubViewModel['actions'] => {
  const admin = role === 'admin';

  return {
    add: [
      ...(capabilities.addService
        ? [
            {
              label: M.monitor,
              description: M.monitorDescription,
              to: MONGO_PATHS.addService,
              icon: AddLinkOutlined,
            },
          ]
        : []),
      ...(capabilities.install
        ? [
            {
              label: M.install,
              description: M.installDescription,
              to: MONGO_PATHS.install,
              icon: DownloadOutlined,
            },
          ]
        : []),
    ],
    more: [
      ...(admin
        ? [
            {
              label: M.manageServices,
              to: MONGO_PATHS.inventoryServices,
              icon: Graph4Icon,
            },
            {
              label: M.configureBackups,
              // With backup management off, the first step is to turn it on.
              to: capabilities.setUpBackup
                ? MONGO_PATHS.backupScheduled
                : MONGO_PATHS.advancedSettings,
              icon: SettingsBackupRestoreOutlined,
            },
            {
              label: M.alertRules,
              to: ALERT_TEMPLATES_PATH,
              icon: NotificationsOutlined,
            },
            {
              label: M.advisors,
              to: MONGO_PATHS.advisors,
              icon: NetworkIntelligenceIcon,
            },
          ]
        : []),
      {
        label: M.documentation,
        to: MONGO_PATHS.documentation,
        icon: ArrowOutwardOutlined,
        external: true,
      },
    ],
  };
};

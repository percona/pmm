// The module, not the package index, which would load the whole Operations UI.
import { formatAge } from '@pmm/plugins-om/src/format';
import { ExtensionIcon } from 'lib/peak-ui-pending-icons';
import type {
  HubApp,
  HubCapabilities,
  HubDoor,
  HubStatus,
} from 'hubs/template/hub.types';
import { MONGO_PATHS } from './mongodb.doors';
import { Messages } from './mongodb.messages';
import type { MongoOperationsInput } from './mongodb.types';

export interface AppsInput {
  capabilities: Pick<HubCapabilities, 'operations' | 'extensions'>;
  /** No MongoDB service shown. */
  emptyEstate: boolean;
  /** Null when Operations is off or its data failed. */
  operations: MongoOperationsInput | null;
  backups: {
    featureOff: boolean;
    notSetUp: boolean;
    schedules: number;
    failedRecently: number;
    lastGoodBackup: HubStatus['lastGoodBackup'];
  };
  now: number;
}

const page = (label: string, to: string): HubDoor => ({ label, to });

const operationsSummary = (
  operations: MongoOperationsInput | null,
  emptyEstate: boolean,
  now: number
): HubApp['summary'] => {
  if (emptyEstate) {
    return { text: Messages.apps.operationsEmpty };
  }
  if (!operations) {
    return undefined;
  }
  const up: HubDoor = {
    label: Messages.apps.operationsUp(
      operations.summary.up,
      operations.summary.services
    ),
    to: MONGO_PATHS.operationsServices,
  };
  const scan: HubDoor = {
    label: operations.summary.lastScanAt
      ? Messages.apps.lastScan(formatAge(operations.summary.lastScanAt, now))
      : Messages.apps.noScanYet,
    to: MONGO_PATHS.operationsScans,
  };

  return {
    text: Messages.where([up.label, scan.label]),
    doors: [up, scan],
  };
};

const backupsSummary = (
  {
    featureOff,
    notSetUp,
    schedules,
    failedRecently,
    lastGoodBackup,
  }: AppsInput['backups'],
  now: number
): HubApp['summary'] => {
  if (featureOff) {
    return { text: Messages.apps.backupsOff };
  }
  if (lastGoodBackup === 'unknown') {
    return { text: Messages.apps.backupsUnknown };
  }
  if (notSetUp) {
    return { text: Messages.apps.backupsNotSetUp };
  }
  const scheduled = page(
    Messages.apps.schedules(schedules),
    MONGO_PATHS.backupScheduled
  );
  const backups = page(
    failedRecently > 0
      ? Messages.apps.failedRecently(failedRecently)
      : typeof lastGoodBackup === 'object'
        ? Messages.apps.lastGood(formatAge(lastGoodBackup.at, now))
        : Messages.apps.noGoodBackup,
    MONGO_PATHS.backupInventory
  );

  return {
    text: Messages.where([scheduled.label, backups.label]),
    doors: [scheduled, backups],
  };
};

/** Operations (admins with it on), MongoDB backups, then Diagnostics with PMM Extensions on; the card title carries Operations' Preview chip. */
export const deriveApps = ({
  capabilities,
  emptyEstate,
  operations,
  backups,
  now,
}: AppsInput): HubApp[] => [
  ...(capabilities.operations
    ? [
        {
          id: 'operations',
          name: Messages.apps.operations,
          icon: ExtensionIcon,
          summary: operationsSummary(operations, emptyEstate, now),
          pages: [
            page(Messages.apps.pages.fleet, MONGO_PATHS.operations),
            page(Messages.apps.pages.nodes, MONGO_PATHS.operationsNodes),
            page(
              Messages.apps.pages.automations,
              MONGO_PATHS.operationsAutomations
            ),
          ],
        },
      ]
    : []),
  {
    id: 'backups',
    name: Messages.apps.backups,
    icon: ExtensionIcon,
    summary: backupsSummary(backups, now),
    pages: [
      page(Messages.apps.pages.inventory, MONGO_PATHS.backupInventory),
      page(Messages.apps.pages.scheduled, MONGO_PATHS.backupScheduled),
      page(Messages.apps.pages.restores, MONGO_PATHS.backupRestores),
      page(Messages.apps.pages.locations, MONGO_PATHS.backupLocations),
    ],
  },
  ...(capabilities.extensions
    ? [
        {
          id: 'diagnostics',
          name: Messages.apps.diagnostics,
          icon: ExtensionIcon,
          summary: { text: Messages.apps.diagnosticsSummary },
          pages: [
            page(Messages.apps.pages.diagnostics, MONGO_PATHS.diagnostics),
          ],
        },
      ]
    : []),
];

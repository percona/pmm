import TourOutlined from '@mui/icons-material/TourOutlined';
import type { CombinedSettings } from 'contexts/settings';
import type { TourName } from 'contexts/tour';
import {
  NAV_ADVISORS,
  NAV_CONFIGURATION,
} from 'contexts/navigation/navigation.constants';
import {
  PMM_NEW_NAV_GRAFANA_PATH,
  PMM_NEW_NAV_UPDATES_PATH,
  PMM_NEW_NAV_PATH,
  PMM_SERVICENOW_SETTINGS_PATH,
} from 'lib/constants';
import { ExtensionIcon } from 'lib/peak-ui-pending-icons';
import { CARD_IDS, getCardData } from 'pages/help-center/HelpCenter.constants';
import { Messages as SettingsMessages } from 'pages/settings/Settings.messages';
import { DistributionMethod, type VersionResponse } from 'types/version.types';
import { searchItem } from '../search.keywords';
import { Messages } from '../search.messages';
import type { SearchItem } from '../search.types';

export interface StaticSourceContext {
  isAdmin: boolean;
  isEditor: boolean;
  settings?: CombinedSettings | null;
  version?: VersionResponse;
  startTour: (tourName: TourName) => void;
}

const SETTINGS_PATH = `${PMM_NEW_NAV_PATH}/settings`;
const ADVANCED_SETTINGS_PATH = `${SETTINGS_PATH}/advanced-settings`;

const SETTINGS_CRUMBS = [
  Messages.crumbs.configuration,
  Messages.crumbs.settings,
];

// A feature counts as on unless the server says it is off.
const isFeatureOn = (flag?: boolean) => flag !== false;

const page = (
  slug: string,
  label: string,
  to: string,
  description: string[],
  navIcon?: SearchItem['navIcon']
) =>
  searchItem({
    id: `page:${slug}`,
    kind: 'page',
    group: 'pages',
    label,
    description,
    navIcon,
    to,
  });

export const extraPages = ({
  isAdmin,
  settings,
}: StaticSourceContext): SearchItem[] => {
  const { pages: labels, crumbs } = Messages;
  const items = [
    page(
      'client-updates',
      labels.clientUpdates,
      `${PMM_NEW_NAV_UPDATES_PATH}/clients`,
      [crumbs.configuration, crumbs.updates],
      NAV_CONFIGURATION.icon
    ),
  ];

  if (isFeatureOn(settings?.advisorEnabled)) {
    items.push(
      ...(
        [
          ['configuration', labels.configurationAdvisors],
          ['performance', labels.performanceAdvisors],
          ['query', labels.queryAdvisors],
          ['security', labels.securityAdvisors],
        ] as const
      ).map(([category, label]) =>
        page(
          `advisors-${category}`,
          label,
          `${PMM_NEW_NAV_GRAFANA_PATH}/advisors/${category}`,
          [crumbs.analysis, crumbs.advisors],
          NAV_ADVISORS.icon
        )
      )
    );
  }

  if (isFeatureOn(settings?.backupManagementEnabled)) {
    items.push(
      ...(
        [
          ['backup-inventory', labels.backupInventory, 'inventory'],
          ['backup-scheduled', labels.scheduledBackups, 'scheduled'],
          ['backup-restore', labels.restores, 'restore'],
          ['backup-locations', labels.storageLocations, 'locations'],
        ] as const
      ).map(([slug, label, path]) =>
        page(
          slug,
          label,
          `${PMM_NEW_NAV_GRAFANA_PATH}/backup/${path}`,
          [crumbs.apps, crumbs.mongodbBackups],
          ExtensionIcon
        )
      )
    );
  }

  if (isAdmin) {
    items.push(
      page(
        'plugins',
        labels.plugins,
        `${PMM_NEW_NAV_GRAFANA_PATH}/plugins`,
        [crumbs.configuration],
        NAV_CONFIGURATION.icon
      ),
      page(
        'data-sources',
        labels.dataSources,
        `${PMM_NEW_NAV_GRAFANA_PATH}/connections/datasources`,
        [crumbs.configuration],
        NAV_CONFIGURATION.icon
      )
    );
  }

  return items;
};

const setting = (
  fields: Pick<SearchItem, 'id' | 'label' | 'to'> &
    Partial<Pick<SearchItem, 'keywords' | 'description' | 'dedupeKey'>>
) =>
  searchItem({
    kind: 'setting',
    group: 'settings',
    description: SETTINGS_CRUMBS,
    ...fields,
  });

export const settingsItems = ({
  isAdmin,
  settings,
  version,
}: StaticSourceContext): SearchItem[] => {
  if (!isAdmin) {
    return [];
  }

  const { tabs, advanced } = SettingsMessages;
  const tabItems = [
    setting({
      id: 'setting:metrics-resolution',
      label: tabs.metrics,
      to: `${SETTINGS_PATH}/metrics-resolution`,
      keywords: ['resolution', 'scrape interval', 'low', 'medium', 'high'],
    }),
    setting({
      id: 'setting:advanced-settings',
      label: tabs.advanced,
      to: ADVANCED_SETTINGS_PATH,
    }),
  ];

  if (version?.distributionMethod === DistributionMethod.ami) {
    tabItems.push(
      setting({
        id: 'setting:ssh-key',
        label: tabs.ssh,
        to: `${SETTINGS_PATH}/ssh-key`,
      })
    );
  }

  if (settings?.extensionsEnabled === true) {
    tabItems.push(
      setting({
        id: 'setting:servicenow-connection',
        label: tabs.serviceNow,
        to: PMM_SERVICENOW_SETTINGS_PATH,
      })
    );
  }

  const fieldItems = (
    [
      ['data-retention', advanced.retentionLabel, ['retention', 'storage']],
      ['public-address', advanced.publicAddressLabel, ['url', 'hostname']],
      ['telemetry', advanced.telemetryLabel, []],
      ['check-for-updates', advanced.updatesLabel, ['updates']],
      [
        'advisor-run-intervals',
        Messages.settings.advisorRunIntervals,
        ['rare', 'standard', 'frequent'],
      ],
      ['azure-monitoring', advanced.azureDiscoverLabel, ['azure']],
      [
        'access-control',
        advanced.accessControl,
        ['lbac', 'label based access', 'roles'],
      ],
      [
        'feature-management',
        advanced.featureManagementLabel,
        [
          'alerting',
          'backup management',
          'qan for pmm server',
          'operations for mongodb',
        ],
      ],
    ] as const
  ).map(([slug, label, keywords]) => {
    const id = `setting:${slug}`;

    return setting({
      id,
      label,
      to: ADVANCED_SETTINGS_PATH,
      keywords: [...keywords],
      description: [...SETTINGS_CRUMBS, Messages.crumbs.advancedSettings],
      // Each field stays its own result, though all of them open the same tab.
      dedupeKey: id,
    });
  });

  return [...tabItems, ...fieldItems];
};

export const actionItems = ({
  isAdmin,
  isEditor,
  settings,
  startTour,
}: StaticSourceContext): SearchItem[] => {
  const { actions: labels } = Messages;
  const action = (
    slug: string,
    label: string,
    fields: Partial<
      Pick<SearchItem, 'keywords' | 'navIcon' | 'perform' | 'to'>
    > = {}
  ) =>
    searchItem({
      id: `action:${slug}`,
      kind: 'action',
      group: 'actions',
      label,
      ...fields,
    });
  const items = [
    action('start-tour', labels.startTour, {
      keywords: ['tips', 'tour', 'onboarding'],
      navIcon: TourOutlined,
      perform: () => startTour('product'),
    }),
  ];

  if (isEditor) {
    items.push(
      action('create-alert-rule', labels.createAlertRule, {
        to: `${PMM_NEW_NAV_GRAFANA_PATH}/alerting/new/alerting`,
      }),
      action('create-dashboard', labels.createDashboard, {
        to: `${PMM_NEW_NAV_GRAFANA_PATH}/dashboard/new`,
      }),
      action('import-dashboard', labels.importDashboard, {
        to: `${PMM_NEW_NAV_GRAFANA_PATH}/dashboard/import`,
      })
    );
  }

  if (isAdmin && isFeatureOn(settings?.backupManagementEnabled)) {
    items.push(
      action('create-backup', labels.createBackup, {
        to: `${PMM_NEW_NAV_GRAFANA_PATH}/backup/new`,
      })
    );
  }

  if (isAdmin) {
    items.push(
      action('create-pmm-dump', labels.createPmmDump, {
        to: `${PMM_NEW_NAV_GRAFANA_PATH}/pmm-dump/new`,
      })
    );
  }

  return items;
};

export const helpItems = ({
  isAdmin,
  startTour,
}: StaticSourceContext): SearchItem[] =>
  getCardData({ startProductTour: () => startTour('product') })
    .filter(
      (card) => card.id !== CARD_IDS.tips && !(card.adminOnly && !isAdmin)
    )
    .flatMap((card) => {
      const [button] = card.buttons;
      const label =
        card.id === CARD_IDS.pmmLogs ? Messages.help.downloadLogs : card.title;
      const fields = {
        id: `help:${card.id}`,
        kind: 'help' as const,
        group: 'help' as const,
        label,
      };

      if (button?.to) {
        return [
          searchItem({
            ...fields,
            description: Messages.help.description,
            to: button.to,
          }),
        ];
      }

      if (button?.url) {
        const external =
          button.target === '_blank' || button.url === '/logs.zip';

        return [
          searchItem({
            ...fields,
            description: external
              ? Messages.help.externalDescription
              : Messages.help.description,
            href: button.url,
            target: external ? '_blank' : '_self',
            external,
          }),
        ];
      }

      return [];
    });

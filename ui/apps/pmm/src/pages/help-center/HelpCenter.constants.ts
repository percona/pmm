import { PMM_NEW_NAV_GRAFANA_PATH } from 'lib/constants';
import { HelpCard } from './help-center-card/HelpCenterCard.types';

export const CARD_IDS = {
  pmmDocs: 'pmm-docs',
  support: 'support',
  forum: 'forum',
  pmmDump: 'pmm-dump',
  pmmLogs: 'pmm-logs',
  tips: 'tips',
};

export const START_ICON = {
  download: 'download',
  map: 'map',
};

export const CARDS_DATA: HelpCard[] = [
  {
    id: CARD_IDS.pmmDocs,
    title: 'PMM Documentation',
    description:
      'From setup to troubleshooting, you’ll find step-by-step instructions, tips, and best practices to get the most out of PMM.',
    buttons: [
      {
        text: 'View docs',
        target: '_blank',
        url: 'https://per.co.na/pmm_documentation',
      },
    ],
    adminOnly: false,
    borderColor: '#1486FF',
  },
  {
    id: CARD_IDS.support,
    title: 'Get Percona Support',
    description:
      'From 24/7 technical support to fully managed services, Percona’s trusted experts are ready to help you optimize, troubleshoot, and scale.',
    buttons: [
      {
        text: 'Contact Support',
        target: '_blank',
        url: 'https://per.co.na/pmm_support',
      },
    ],
    adminOnly: false,
    borderColor: '#F24500',
  },
  {
    id: CARD_IDS.forum,
    title: 'Percona Forum',
    description:
      'A friendly space to connect with other users, share insights, and get answers from the community and from the Percona experts.',
    buttons: [
      {
        text: 'View forum',
        target: '_blank',
        url: 'https://per.co.na/PMM3_forum',
      },
    ],
    adminOnly: false,
    borderColor: '#30D1B2',
  },
  {
    id: CARD_IDS.pmmDump,
    title: 'PMM Dump',
    description:
      'Generate datasets to securely share your data with Percona Support. This helps our experts quickly diagnose and replicate issues.',
    buttons: [
      {
        text: 'Manage datasets',
        to: `${PMM_NEW_NAV_GRAFANA_PATH}/pmm-dump`,
      },
    ],
    adminOnly: true,
    borderColor: '#F0B336',
  },
  {
    id: CARD_IDS.pmmLogs,
    title: 'PMM Logs',
    description:
      'Download your PMM logs as a ZIP file for easy sharing and faster issue diagnosis.',
    buttons: [
      {
        text: 'Export logs',
        target: '_blank',
        url: '/logs.zip',
        startIconName: START_ICON.download,
      },
    ],
    adminOnly: true,
  },
  {
    id: CARD_IDS.tips,
    title: 'Useful Tips',
    description:
      'Need a refresher? Access the onboarding tour tips or the keyboard shortcuts.',
    adminOnly: false,
    buttons: [
      {
        text: 'Start PMM tour',
        startIconName: START_ICON.map,
      },
    ],
  },
];

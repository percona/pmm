export const cardIds = {
  pmmDocs: 'pmm-docs',
  support: 'support',
  forum: 'forum',
  pmmDump: 'pmm-dump',
  pmmLogs: 'pmm-logs',
  tips: 'tips',
};

export const startIcon = {
  download: 'download',
  map: 'map',
};

export const CardsData = [
  {
    id: cardIds.pmmDocs,
    title: 'PMM Documentation',
    description:
      'From setup to troubleshooting, you’ll find step-by-step instructions, tips, and best practices to get the most out of PMM.',
    buttons: [
      {
        buttonText: 'View docs',
        target: '_blank',
        url: 'https://per.co.na/pmm_documentation',
        startIconName: '',
      },
    ],
    borderColor: '#1486FF',
  },
  {
    id: cardIds.support,
    title: 'Get Percona Support',
    description:
      'From 24/7 technical support to fully managed services, Percona’s trusted experts are ready to help you optimize, troubleshoot, and scale.',
    buttons: [
      {
        buttonText: 'Contact Support',
        target: '_blank',
        url: 'https://per.co.na/pmm_support',
        startIconName: '',
      },
    ],
    borderColor: '#F24500',
  },
  {
    id: cardIds.forum,
    title: 'Percona Forum',
    description:
      'A friendly space to connect with other users, share insights, and get answers from the community and from the Percona experts.',
    buttons: [
      {
        buttonText: 'View forum',
        target: '_blank',
        url: 'https://per.co.na/PMM3_forum',
        startIconName: '',
      },
    ],
    borderColor: '#30D1B2',
  },
  {
    id: cardIds.pmmDump,
    title: 'PMM Dump',
    description:
      'Generate datasets to securely share your data with Percona Support. This helps our experts quickly diagnose and replicate issues.',
    buttons: [
      {
        buttonText: 'Manage datasets',
        target: '',
        url: '/graph/pmm-dump',
        startIconName: '',
      },
    ],
    borderColor: '#F0B336',
  },
  {
    id: cardIds.pmmLogs,
    title: 'PMM Logs',
    description:
      'Download your PMM logs as a ZIP file for easy sharing and faster issue diagnosis.',
    buttons: [
      {
        buttonText: 'Export logs',
        target: '_blank',
        url: '/logs.zip',
        startIconName: startIcon.download,
      },
    ],
  },
  {
    id: cardIds.tips,
    title: 'Useful Tips',
    description:
      'Need a refresher? Access the onboarding tour tips or the keyboard shortcuts.',
    buttons: [
      {
        buttonText: 'Start PMM tour',
        startIconName: startIcon.map,
        target: '',
        url: '',
      },
      {
        buttonText: 'Shortcuts',
        startIconName: '',
        target: '',
        url: 'https://per.co.na/pmm_documentation',
      },
    ],
  },
];

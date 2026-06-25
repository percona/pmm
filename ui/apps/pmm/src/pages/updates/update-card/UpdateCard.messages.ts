export const Messages = {
  fetchError: "Couldn't load current version information.",
  upToDate: 'This PMM instance is up to date.',
  newUpdateAvailable: (version: string) =>
    `New update available: PMM ${version}`,
  runningVersion: 'Running version:',
  newVersion: 'New version:',
  lastChecked: 'Last checked:',
  home: 'PMM home',
  checkNow: 'Check updates now',
  checking: 'Checking',
  howToUpdateDocs: 'How to update docs',
  error: 'There was a problem during the update',

  deprecation: {
    heading: 'UI upgrades deprecated',
    paragraph1BeforeUpdateNow: ': This ',
    paragraph1AfterUpdateNow: ' button will be removed in PMM 3.9.0.',
    viaIntro: 'After that, PMM upgrades will only be available via\u00a0',
    docker: 'Docker',
    afterDocker: ' (recommended), ',
    podman: 'Podman',
    afterPodman: ', or ',
    helm: 'Helm',
    afterHelm: '.',
    reminder: 'Switch before then to keep upgrading PMM to newer versions.',
  },
};

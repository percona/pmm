const plural = (count: number, one: string, many: string) =>
  `${count} ${count === 1 ? one : many}`;

export const Messages = {
  createTitle: 'New schedule',
  editTitle: 'Edit schedule',
  close: 'Close',
  cancel: 'Cancel',
  create: 'Create schedule',
  save: 'Save changes',
  name: {
    label: 'Name',
    placeholder: 'For example: Prod security, weekdays',
    help: 'Shown in Run history and in the email subject.',
  },
  enabled: {
    label: 'Enabled',
    help: "Paused schedules keep their settings but don't run.",
  },
  when: {
    title: 'When',
    days: 'Days',
    setTo: 'Set to',
    everyDay: 'every day',
    weekdays: 'weekdays',
    weekends: 'weekends',
    time: 'Time (UTC)',
    timeHelp: 'All schedules use UTC, like backups.',
    previewEmpty:
      'Pick at least one day and a time to see when this schedule runs.',
    runs: (days: string, time: string) => `Runs ${days} at ${time} UTC`,
    local: (time: string, zone: string, offset: string) =>
      `, ${time} in your time zone (${zone}, ${offset})`,
    nextRuns: 'Next runs',
    next: 'Next',
    yourTime: (when: string) => ` · ${when} your time`,
    shift: (next: boolean) =>
      `In your time zone this lands on the ${next ? 'next' : 'previous'} day. Days are picked in UTC.`,
    dst: (zone: string) =>
      `UTC has no daylight saving time, so the time in ${zone} moves by an hour when your clocks change.`,
  },
  checks: {
    title: 'Checks',
    all: 'All checks',
    allHint: (total: number, disabled: number) =>
      disabled
        ? `${plural(total, 'check', 'checks')} today, ${disabled} of them disabled in the Catalog. Checks added later are included.`
        : `${plural(total, 'check', 'checks')} today. Checks added later are included.`,
    selected: 'Selected checks',
    selectedHint:
      'Only the checks you pick. New checks are not added automatically.',
    search: 'Search checks',
    allGroups: 'All categories',
    group: 'Category',
    disabledInCatalog: 'Disabled in Catalog',
    disabledTooltip: 'Skipped while disabled in the Catalog',
    noMatch: 'No check matches these filters.',
  },
  services: {
    title: 'Services',
    all: 'All services',
    allHint: (total: number) =>
      `${plural(total, 'service', 'services')} today. Services added later are included.`,
    selected: 'Selected services',
    selectedHint:
      'Only the services you pick. Removed services drop out of the schedule.',
    search: 'Search services, clusters, nodes',
    allGroups: 'All environments',
    group: 'Environment',
    noMatch: 'No service matches these filters.',
  },
  picker: {
    selectShown: (n: number) => `Select the ${n} shown`,
    clear: 'Clear selection',
    onlySelected: 'Show selected only',
    count: (selected: number, total: number) =>
      `${selected} of ${total} selected`,
  },
  coverage: {
    title: 'What each run covers',
    hint: 'Checks run only on services of their technology. Checks disabled in the Catalog are skipped.',
    technology: 'Technology',
    checks: 'Checks',
    services: 'Services',
    executions: 'Check executions',
    eachRun: 'Each run',
    nothing: (technologies: string) =>
      `None of the selected checks applies to the selected services, so every run would have nothing to do. Pick checks for ${technologies}, or other services.`,
    yourServices: 'your services',
    idle: (checks: string, technologies: string, all: boolean) =>
      `${checks} won't run: ${all ? 'there is no' : 'no'} ${technologies} service${all ? ' in Inventory' : ' is selected'}.`,
    unchecked: (services: string, technologies: string, one: boolean) =>
      `${services} ${one ? 'gets' : 'get'} no checks: no ${technologies} check is selected.`,
    checkCount: (n: number, technology: string) =>
      `${n} ${technology} ${n === 1 ? 'check' : 'checks'}`,
    serviceCount: (n: number, technology: string) =>
      `${n} ${technology} ${n === 1 ? 'service' : 'services'}`,
  },
  email: {
    title: 'Email report',
    notificationsOff:
      "Email notifications are turned off in Settings → Advisors, so no schedule sends email until they're turned on. You can still set recipients now.",
    settings: 'Settings recipients',
    settingsHint:
      'The default list in Settings → Advisors. Changes there apply here too.',
    noSettingsRecipients: 'No default recipients are set.',
    changeInSettings: 'Change in Settings',
    custom: 'These recipients',
    customHint: "Only these addresses get this schedule's reports.",
    addresses: 'Email addresses',
    addressesPlaceholder: 'secops@example.com, dba@example.com',
    addressesHelp: 'Separate addresses with commas or spaces. Up to 20.',
    addressesCount: (n: number) =>
      `${plural(n, 'address', 'addresses')}. Separate addresses with commas or spaces. Up to 20.`,
    off: "Don't email",
    offHint: 'Results stay in Insights and Run history.',
    severities: 'Severities',
    sameAsSettings: 'Same as Settings',
    sameAsSettingsHint: (list: string) =>
      list
        ? `${list}, as set in Settings → Advisors. Changes there apply here too.`
        : 'The default severities in Settings → Advisors. Changes there apply here too.',
    onlyThese: 'Only these',
    onlyTheseHint:
      'For this schedule only. A run with no finding at these severities sends no email.',
    severitiesGroup: 'Severities to email',
    test: 'Send test email',
    testPending: 'Sending…',
    testHint:
      "Sends a sample report to this schedule's recipients for the chosen severities, unsaved changes included. No checks are run.",
    testDisabled: 'Enter valid recipients and pick at least one severity first',
    testSent: (recipients: string, severities: string) =>
      `Sent a test report to ${recipients} with sample ${severities} findings. No checks were run.`,
  },
  validation: {
    name: 'Give the schedule a name.',
    nameTaken: (name: string) => `A schedule named "${name}" already exists.`,
    days: 'Pick at least one day.',
    time: 'Enter a time as HH:MM.',
    checks: 'Pick at least one check, or choose All checks.',
    services: 'Pick at least one service, or choose All services.',
    emailRequired: 'Enter at least one address, or choose Settings recipients.',
    emailInvalid: (list: string) => `Not a valid address: ${list}`,
    emailMax: (max: number, count: number) =>
      `Up to ${max} addresses; there are ${count}.`,
    severities: 'Pick at least one severity, or choose Same as Settings.',
    fix: (fields: string[]) =>
      `Fix the highlighted ${fields.length > 1 ? 'fields' : 'field'}: ${fields.join(', ')}.`,
  },
  success: {
    created: (name: string) => `Created "${name}".`,
    saved: (name: string) => `Saved "${name}".`,
    next: (when: string) => `Next run: ${when}.`,
    paused: 'It stays paused until you enable it.',
  },
};

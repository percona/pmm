const plural = (count: number, one: string, many: string) =>
  `${count} ${count === 1 ? one : many}`;

export const Messages = {
  title: 'Advisor schedules',
  description:
    'When Advisor checks run, which checks and services each run covers, and who gets the email report. Runs never overlap: a schedule that falls due during another run starts as soon as that run ends.',
  newSchedule: 'New schedule',
  reload: 'Reload schedules',
  allTimesAre: 'All times are',
  yourTimeZone: (zone: string, offset: string) =>
    `Your time zone: ${zone} (${offset})`,
  settingsLink: 'Settings → Advisors',
  banners: {
    advisorsOff:
      'Advisors are turned off, so no schedule runs. Schedules are kept. Turn Advisors on in',
    notificationsOff:
      'Advisor email notifications are turned off, so no schedule sends email. Recipients below are kept. Turn notifications on in',
    aRun: 'An Advisor run',
    running: (done: number, planned: number, started: string) =>
      `is running: ${done} of ${planned} checks done, started ${started}. Schedules that fall due now wait and start when it ends.`,
    queued:
      'is queued and starts within a minute. Schedules that fall due now wait and start when it ends.',
    openRun: 'Open in Run history',
  },
  week: {
    title: 'This week, UTC',
    noSchedules:
      'Nothing is scheduled. Advisor checks run only when someone starts them.',
    advisorsOff: 'Nothing runs while Advisors are turned off.',
    now: 'now',
    mark: (
      name: string,
      when: string,
      local: string,
      checks: number,
      services: number
    ) =>
      `${name}: ${when}${local ? ` (${local} your time)` : ''}, ${plural(checks, 'check', 'checks')} on ${plural(services, 'service', 'services')}`,
  },
  columns: {
    schedule: 'Schedule',
    when: 'When',
    checks: 'Checks',
    services: 'Services',
    email: 'Email',
    nextRun: 'Next run',
    lastRun: 'Last run',
    enabled: 'Enabled',
    actions: 'Actions',
  },
  days: {
    everyDay: 'Every day',
    weekdays: 'Mon–Fri',
    weekend: 'Sat, Sun',
  },
  phrases: {
    everyDay: 'every day',
    everyWeekday: 'every weekday',
    everyWeekend: 'every Saturday and Sunday',
    every: (days: string) => `every ${days}`,
  },
  localTime: (time: string) => `${time} your time`,
  nextDay: 'next day',
  dayBefore: 'day before',
  relative: {
    minutes: (n: number) => `${n} min`,
    hours: (n: number) => `${n} h`,
    days: (n: number) => plural(n, 'day', 'days'),
    in: (text: string) => `in ${text}`,
    ago: (text: string) => `${text} ago`,
  },
  allChecks: 'All checks',
  allServices: 'All services',
  checksCount: (n: number) => plural(n, 'check', 'checks'),
  servicesCount: (n: number) => plural(n, 'service', 'services'),
  servicesOf: (n: number, total: number) =>
    `${n} of ${plural(total, 'service', 'services')}`,
  today: (n: number) => `${n} today`,
  targetsNoService: 'Targets no existing service',
  targetsNoCheck: 'Targets no existing check',
  targetsNoServiceTooltip:
    'Every selected service was removed, so runs have nothing to do. Edit the schedule to pick services.',
  targetsNoCheckTooltip:
    'Every selected check was removed, so runs have nothing to do. Edit the schedule to pick checks.',
  email: {
    none: 'No email',
    settingsRecipients: 'Settings recipients',
    ownRecipients: 'Own recipients',
    addresses: (n: number) => plural(n, 'address', 'addresses'),
    more: (n: number) => `+${n}`,
    notificationsOff: 'Email notifications are off in Settings',
  },
  allSeverities: 'all severities',
  severityOnly: (severity: string) => `${severity} only`,
  next: {
    notScheduled: 'Not scheduled',
    advisorsOff: 'Advisors are off',
    paused: 'Paused',
  },
  last: {
    never: 'Never run',
    findings: (n: number) => plural(n, 'finding', 'findings'),
    openRun: 'Open this run in Run history',
    running: (done: number, planned: number) => `Running · ${done}/${planned}`,
    queued: 'Queued',
    interrupted: 'Interrupted',
    aborted: 'Aborted',
    started: (when: string) => `started ${when}`,
  },
  runStatus: {
    queued: 'Queued',
    running: 'Running',
    completed: 'Completed',
    interrupted: 'Interrupted',
    aborted: 'Aborted',
  },
  toggle: {
    pause: 'Pause this schedule',
    enable: 'Enable this schedule',
    label: (name: string) => `Enabled: ${name}`,
  },
  actions: {
    more: (name: string) => `More actions for ${name}`,
    runNow: 'Run now',
    edit: 'Edit',
    duplicate: 'Duplicate',
    viewRuns: 'View runs',
    delete: 'Delete',
  },
  empty: {
    title: 'No schedules',
    body: 'Advisor checks now run only when someone starts them from the Catalog. Create a schedule to run them automatically, or restore the default: every day at 06:00 UTC, all checks on all services, emailed to the Settings recipients.',
    restore: 'Create the default schedule',
  },
  deleteDialog: {
    title: (name: string) => `Delete "${name}"?`,
    last: 'This is the last schedule. Without it, Advisor checks run only when someone starts them.',
    notLast: 'Its future runs are cancelled.',
    history: 'Past runs stay in Run history under this name.',
    cancel: 'Cancel',
    confirm: 'Delete',
  },
  success: {
    started: (name: string) => `Started "${name}". Follow it in Run history.`,
    enabled: (name: string, next: string) =>
      `Enabled "${name}". Next run: ${next}.`,
    paused: (name: string) =>
      `Paused "${name}". It won't run until you enable it.`,
    duplicated: (name: string) =>
      `Duplicated as "${name}". It stays paused until you enable it, so nothing runs twice.`,
    deleted: (name: string) => `Deleted "${name}".`,
    created: (name: string, next: string) =>
      `Created "${name}". Next run: ${next}.`,
  },
  pane: {
    close: 'Close details',
    paused: 'Paused',
    when: 'When',
    checks: 'Checks',
    services: 'Services',
    email: 'Email',
    whenValue: (days: string, time: string) => `${days} at ${time} UTC`,
    allChecks: (n: number) => `All, ${n} today, including checks added later`,
    allServices: (n: number) =>
      `All, ${n} today, including services added later`,
    selected: (n: number) => `${n} selected`,
    missing: (n: number) => `${n} no longer ${n === 1 ? 'exists' : 'exist'}`,
    settingsRecipients: (list: string) => `Settings recipients: ${list}`,
    noSettingsRecipients: 'Settings recipients: none set',
    ownSeverities: (list: string) => `${list} findings, set on this schedule`,
    settingsSeverities: (list: string) => `${list} findings, as in Settings`,
    nextRuns: 'Next runs',
    nothingWhileOff: 'Nothing runs while Advisors are turned off.',
    pausedNote: 'Paused. Enable the schedule to plan its runs.',
    yourTime: (time: string) => `${time} your time`,
    coverage: 'What each run covers',
    service: 'Service',
    technology: 'Technology',
    environment: 'Environment',
    cluster: 'Cluster',
    checksColumn: 'Checks',
    coverageTotal: (checks: number, services: number) =>
      `${plural(checks, 'check', 'checks')} on ${plural(services, 'service', 'services')}`,
    coverageNote:
      'Checks run only on services of their technology. Checks disabled in the Catalog, for all services or for one, are skipped.',
    noCheckApplies: (names: string) => `No selected check applies to ${names}.`,
    noServices: 'No service is in scope.',
    checksByTechnology: (n: number) => plural(n, 'check', 'checks'),
    noMatchingService: 'no matching service',
    recentRuns: 'Recent runs',
    notRunYet: 'This schedule has not run yet.',
    open: 'Open',
  },
};

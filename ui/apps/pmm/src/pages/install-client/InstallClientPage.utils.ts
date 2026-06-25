// IMPORTANT: keep this list in sync with `SUPPORTED_TECHNOLOGIES` in
// api/installToken.ts — adding a tech here without adding it there gets you
// a client-side "unsupported technology" error when generating a token.
export type Technology = 'mysql' | 'postgresql' | 'mongodb' | 'valkey';
export type CredentialsMode = 'prompt' | 'env' | 'flags';

/** MySQL QAN sources — keep in sync with pmm-admin add mysql --query-source. */
export type MySQLQuerySource = '' | 'slowlog' | 'perfschema' | 'none';

export const MYSQL_QUERY_SOURCES: { value: MySQLQuerySource; label: string }[] = [
  { value: '', label: 'Default (slowlog)' },
  { value: 'slowlog', label: 'Slow log' },
  { value: 'perfschema', label: 'Performance Schema' },
  { value: 'none', label: 'None (metrics only)' },
];

/** Default listen ports — keep in sync with install-pmm-client.sh add_* defaults. */
export const DEFAULT_DB_PORTS: Record<Technology, number> = {
  mysql: 3306,
  postgresql: 5432,
  mongodb: 27017,
  valkey: 6379,
};

/**
 * Suggested PMM service name when the field is left empty in the wizard.
 */
export function suggestDbServiceName(
  technology: Technology,
  dbPort: string,
  nodeName = '',
): string {
  const hostLabel = nodeName.trim() || '<hostname>';
  const base = `${hostLabel}-${technology}`;
  const port = dbPort.trim();
  if (!port) {
    return base;
  }
  const portNum = Number(port);
  if (!Number.isInteger(portNum) || portNum <= 0 || portNum > 65535) {
    return base;
  }
  return `${base}-${port}`;
}

/**
 * Formats the remaining lifetime of an install token as MM:SS.
 * Negative inputs (already expired) are clamped to "0:00" so callers can
 * branch on isExpired separately without seeing odd negative timers.
 */
export const formatExpiresIn = (secondsLeft: number): string => {
  // Guard against NaN/Infinity (e.g. an Invalid Date upstream) so the chip never
  // renders "NaN:NaN" and stays stuck; treat anything non-finite as expired.
  const safe = Number.isFinite(secondsLeft) ? Math.max(0, Math.floor(secondsLeft)) : 0;
  const minutes = Math.floor(safe / 60);
  const seconds = safe % 60;
  return `${minutes}:${seconds.toString().padStart(2, '0')}`;
};

/**
 * Normalizes PMM host input to hostname or hostname:port.
 * Strips protocol and path when the user pastes a full URL.
 */
const sanitizePmmHost = (input: string): string => {
  const trimmed = input.trim();
  if (!trimmed) {
    return '';
  }

  // Strip with plain string ops rather than `new URL()` so an explicitly typed default
  // port (":443") is preserved instead of being normalized away. Order: scheme, a
  // leading protocol-relative "//", any path/query/fragment, then any userinfo
  // ("user:pass@") — leaving just hostname[:port] with a single authority.
  return trimmed
    .replace(/^https?:\/\//i, '')
    .replace(/^\/\//, '')
    .split(/[/?#]/)[0]
    .replace(/^.*@/, '')
    .trim();
};

/**
 * Builds PMM_SERVER_URL for install scripts. Token is percent-encoded in the userinfo.
 * `pmmHost` is hostname or hostname:port (defaults to current page host when empty).
 */
export function buildPmmServerURL(pmmHost: string, token: string): string {
  const authority =
    sanitizePmmHost(pmmHost) ||
    (typeof window !== 'undefined' ? window.location.host : 'localhost');
  const t = token.trim();
  if (!t) {
    return `https://service_token:<TOKEN>@${authority}`;
  }
  return `https://service_token:${encodeURIComponent(t)}@${authority}`;
}

export interface InstallCommandOptions {
  installerUrl: string;
  technology: Technology;
  credentialsMode: CredentialsMode;
  serverURL: string;
  insecureTLS: boolean;
  registerForce: boolean;
  nodeName: string;
  nodeAddress: string;
  dbUser: string;
  dbPassword: string;
  dbHost: string;
  dbPort: string;
  dbName: string;
  dbAuthDB: string;
  dbServiceName: string;
  /** MySQL only — maps to install-pmm-client.sh --db-query-source */
  dbQuerySource: MySQLQuerySource;
}

export const shellEscape = (value: string): string =>
  `'${value.replace(/'/g, `'\\''`)}'`;

// Where prompt mode tells the user to drop the downloaded script.
const DOWNLOADED_SCRIPT_PATH = '/tmp/install-pmm-client.sh';

const curlDownloadFlags = (insecureTLS: boolean): string =>
  insecureTLS ? '-fsSLk' : '-fsSL';

const appendMysqlQuerySourceFlag = (
  opts: InstallCommandOptions,
  flags: string[]
): void => {
  if (opts.technology !== 'mysql' || !opts.dbQuerySource) {
    return;
  }
  flags.push(`--db-query-source ${shellEscape(opts.dbQuerySource)}`);
};

const appendMysqlQuerySourceEnv = (
  opts: InstallCommandOptions,
  envVars: string[]
): void => {
  if (opts.technology !== 'mysql' || !opts.dbQuerySource) {
    return;
  }
  envVars.push(`DB_QUERY_SOURCE=${shellEscape(opts.dbQuerySource)}`);
};

// renders a two-step "download then sudo -E bash" command
// so the install script gets a real TTY on stdin.
// user only types two things.
const buildPromptModeCommand = (opts: InstallCommandOptions): string => {
  const curl = `curl ${curlDownloadFlags(opts.insecureTLS)} -o ${shellEscape(
    DOWNLOADED_SCRIPT_PATH
  )} ${shellEscape(opts.installerUrl)}`;

  const flags: string[] = [
    `--pmm-server-url ${shellEscape(opts.serverURL)}`,
    `--tech ${shellEscape(opts.technology)}`,
  ];
  if (opts.insecureTLS) {
    flags.push('--pmm-server-insecure-tls');
  }
  if (opts.registerForce) {
    flags.push('--force');
  }
  if (opts.nodeName.trim()) {
    flags.push(`--node-name ${shellEscape(opts.nodeName.trim())}`);
  }
  if (opts.nodeAddress.trim()) {
    flags.push(`--node-address ${shellEscape(opts.nodeAddress.trim())}`);
  }
  if (opts.dbHost.trim()) {
    flags.push(`--db-host ${shellEscape(opts.dbHost.trim())}`);
  }
  if (opts.dbPort.trim()) {
    flags.push(`--db-port ${shellEscape(opts.dbPort.trim())}`);
  }
  if (opts.dbServiceName.trim()) {
    flags.push(`--db-service-name ${shellEscape(opts.dbServiceName.trim())}`);
  }
  if (opts.dbName.trim() && opts.technology === 'postgresql') {
    flags.push(`--db-name ${shellEscape(opts.dbName.trim())}`);
  }
  if (opts.dbAuthDB.trim() && opts.technology === 'mongodb') {
    flags.push(`--db-auth-db ${shellEscape(opts.dbAuthDB.trim())}`);
  }
  appendMysqlQuerySourceFlag(opts, flags);

  // `sudo -E bash <path>` keeps stdin on the caller's TTY (same as plain sudo bash)
  // while preserving the user's environment.
  return [
    curl,
    `sudo -E bash ${shellEscape(DOWNLOADED_SCRIPT_PATH)} \\`,
    `  ${flags.join(' \\\n  ')}`,
  ].join('\n');
};

export const buildInstallCommand = (opts: InstallCommandOptions): string => {
  if (opts.credentialsMode === 'prompt') {
    return buildPromptModeCommand(opts);
  }

  const curl = `curl ${curlDownloadFlags(opts.insecureTLS)} ${shellEscape(opts.installerUrl)}`;

  const envVars: string[] = [
    `PMM_SERVER_URL=${shellEscape(opts.serverURL)}`,
    `TECH=${shellEscape(opts.technology)}`,
  ];

  if (opts.nodeName.trim()) {
    envVars.push(`NODE_NAME=${shellEscape(opts.nodeName.trim())}`);
  }
  if (opts.nodeAddress.trim()) {
    envVars.push(`NODE_ADDRESS=${shellEscape(opts.nodeAddress.trim())}`);
  }

  /** Passed after \`bash -s --\` (matches install-pmm-client.sh). */
  const scriptFlags: string[] = [];
  if (opts.insecureTLS) {
    scriptFlags.push('--pmm-server-insecure-tls');
  }
  if (opts.registerForce) {
    scriptFlags.push('--force');
  }

  if (opts.credentialsMode === 'env') {
    if (opts.dbUser.trim()) {
      envVars.push(`DB_USER=${shellEscape(opts.dbUser.trim())}`);
    }
    if (opts.dbPassword.trim()) {
      envVars.push(`DB_PASSWORD=${shellEscape(opts.dbPassword.trim())}`);
    }
    if (opts.dbHost.trim()) {
      envVars.push(`DB_HOST=${shellEscape(opts.dbHost.trim())}`);
    }
    if (opts.dbPort.trim()) {
      envVars.push(`DB_PORT=${shellEscape(opts.dbPort.trim())}`);
    }
    if (opts.dbServiceName.trim()) {
      envVars.push(`DB_SERVICE_NAME=${shellEscape(opts.dbServiceName.trim())}`);
    }
    if (opts.dbName.trim() && opts.technology === 'postgresql') {
      envVars.push(`DB_NAME=${shellEscape(opts.dbName.trim())}`);
    }
    if (opts.dbAuthDB.trim() && opts.technology === 'mongodb') {
      envVars.push(`DB_AUTH_DB=${shellEscape(opts.dbAuthDB.trim())}`);
    }
    appendMysqlQuerySourceEnv(opts, envVars);
  }

  if (opts.credentialsMode === 'flags') {
    const flags: string[] = [
      `--pmm-server-url ${shellEscape(opts.serverURL)}`,
      `--tech ${shellEscape(opts.technology)}`,
    ];

    if (opts.nodeName.trim()) {
      flags.push(`--node-name ${shellEscape(opts.nodeName.trim())}`);
    }
    if (opts.nodeAddress.trim()) {
      flags.push(`--node-address ${shellEscape(opts.nodeAddress.trim())}`);
    }
    if (opts.insecureTLS) {
      flags.push('--pmm-server-insecure-tls');
    }
    if (opts.registerForce) {
      flags.push('--force');
    }
    if (opts.dbUser.trim()) {
      flags.push(`--db-user ${shellEscape(opts.dbUser.trim())}`);
    }
    if (opts.dbPassword.trim()) {
      flags.push(`--db-password ${shellEscape(opts.dbPassword.trim())}`);
    }
    if (opts.dbHost.trim()) {
      flags.push(`--db-host ${shellEscape(opts.dbHost.trim())}`);
    }
    if (opts.dbPort.trim()) {
      flags.push(`--db-port ${shellEscape(opts.dbPort.trim())}`);
    }
    if (opts.dbServiceName.trim()) {
      flags.push(`--db-service-name ${shellEscape(opts.dbServiceName.trim())}`);
    }
    if (opts.dbName.trim() && opts.technology === 'postgresql') {
      flags.push(`--db-name ${shellEscape(opts.dbName.trim())}`);
    }
    if (opts.dbAuthDB.trim() && opts.technology === 'mongodb') {
      flags.push(`--db-auth-db ${shellEscape(opts.dbAuthDB.trim())}`);
    }
    appendMysqlQuerySourceFlag(opts, flags);

    return [
      `${curl} | sudo -E bash -s -- \\`,
      `  ${flags.join(' \\\n  ')}`,
    ].join('\n');
  }

  const lines: string[] = [`${curl} | sudo -E env \\`];
  envVars.forEach((item) => {
    lines.push(`  ${item} \\`);
  });
  if (scriptFlags.length === 0) {
    lines.push('bash -s --');
  } else {
    lines.push('bash -s -- \\');
    scriptFlags.forEach((flag, index) => {
      const isLast = index === scriptFlags.length - 1;
      lines.push(isLast ? `  ${flag}` : `  ${flag} \\`);
    });
  }
  return lines.join('\n');
};

export type Technology = 'mysql' | 'postgresql' | 'mongodb' | 'valkey';
export type CredentialsMode = 'prompt' | 'env' | 'flags';

/**
 * Builds PMM_SERVER_URL for install scripts. Token is percent-encoded in the userinfo.
 * `pmmHost` is hostname or hostname:port (defaults to current page host when empty).
 */
export function buildPmmServerURL(pmmHost: string, token: string): string {
  const authority =
    pmmHost.trim() ||
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
}

export const shellEscape = (value: string): string =>
  `'${value.replace(/'/g, `'\\''`)}'`;

export const buildInstallCommand = (opts: InstallCommandOptions): string => {
  const curl = `curl -fsSLk ${shellEscape(opts.installerUrl)}`;

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

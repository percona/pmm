import { describe, expect, test } from 'vitest';
import {
  buildInstallCommand,
  buildPmmServerURL,
  DEFAULT_DB_PORTS,
  formatExpiresIn,
  InstallCommandOptions,
  suggestDbServiceName,
} from './InstallClientPage.utils';

// Default to 'env' mode so most existing assertions exercise the curl|bash
// pipeline; prompt-mode tests opt in explicitly via `credentialsMode: 'prompt'`.
// This avoids surprises when shared assertions (curl flags, --pmm-server-insecure-tls)
// happen to also pass in prompt mode but for entirely different reasons.
const baseOptions: InstallCommandOptions = {
  installerUrl: 'https://pmm.example.com/pmm-static/install-pmm-client.sh',
  technology: 'mysql',
  credentialsMode: 'env',
  serverURL: 'https://service_token:GLSA@pmm.example.com:443',
  insecureTLS: true,
  registerForce: false,
  nodeName: '',
  nodeAddress: '',
  dbUser: '',
  dbPassword: '',
  dbHost: '',
  dbPort: '',
  dbName: '',
  dbAuthDB: '',
  dbServiceName: '',
  dbQuerySource: '',
};

const optionsWithDb: InstallCommandOptions = {
  ...baseOptions,
  dbUser: 'pmm',
  dbPassword: 'secret',
  dbHost: '127.0.0.1',
  dbPort: '3306',
  dbServiceName: 'node-mysql',
};

describe('buildPmmServerURL', () => {
  test('uses placeholder when token empty', () => {
    expect(buildPmmServerURL('pmm.example.com:8443', '')).toBe(
      'https://service_token:<TOKEN>@pmm.example.com:8443'
    );
  });

  test('percent-encodes token in userinfo', () => {
    expect(buildPmmServerURL('h:1', 'a:b@c')).toBe(
      'https://service_token:a%3Ab%40c@h:1'
    );
  });

  test('strips protocol from full URLs', () => {
    expect(buildPmmServerURL('https://pmm.example.com:8443', 'tok')).toBe(
      'https://service_token:tok@pmm.example.com:8443'
    );
    expect(buildPmmServerURL('http://pmm.example.com', 'tok')).toBe(
      'https://service_token:tok@pmm.example.com'
    );
  });

  test('strips path and query from pasted URLs', () => {
    expect(buildPmmServerURL('https://pmm.example.com/graph/d/foo', 'tok')).toBe(
      'https://service_token:tok@pmm.example.com'
    );
    expect(buildPmmServerURL('pmm.example.com:443/pmm-ui', 'tok')).toBe(
      'https://service_token:tok@pmm.example.com:443'
    );
  });

  test('strips embedded userinfo so the authority has a single @', () => {
    expect(buildPmmServerURL('user:pw@pmm.example.com:443', 'tok')).toBe(
      'https://service_token:tok@pmm.example.com:443'
    );
    expect(buildPmmServerURL('https://user:pw@pmm.example.com', 'tok')).toBe(
      'https://service_token:tok@pmm.example.com'
    );
  });
});

describe('buildInstallCommand', () => {
  test('env mode renders curl|bash with PMM_SERVER_URL and TECH', () => {
    const cmd = buildInstallCommand(baseOptions);
    expect(cmd).toContain("TECH='mysql'");
    expect(cmd).toContain("PMM_SERVER_URL='https://service_token:GLSA@pmm.example.com:443'");
    expect(cmd).toContain('sudo -E env');
    expect(cmd).toContain('curl -fsSLk');
    expect(cmd).toContain('bash -s --');
    expect(cmd).toContain('--pmm-server-insecure-tls');
    expect(cmd).not.toContain('DB_USER=');
    expect(cmd).not.toContain('DB_PASSWORD=');
  });

  test('omits insecure TLS flag and drops curl -k when disabled', () => {
    const cmd = buildInstallCommand({ ...baseOptions, insecureTLS: false });
    expect(cmd).not.toContain('--pmm-server-insecure-tls');
    expect(cmd).toContain('curl -fsSL ');
    expect(cmd).not.toContain('curl -fsSLk');
    expect(cmd).toContain('bash -s --');
  });

  test('uses curl -fsSLk when insecure TLS is on', () => {
    const cmd = buildInstallCommand({ ...baseOptions, insecureTLS: true });
    expect(cmd).toContain('curl -fsSLk ');
    expect(cmd).toContain('--pmm-server-insecure-tls');
  });

  test('flags mode also respects the insecure TLS toggle for curl', () => {
    const secure = buildInstallCommand({
      ...optionsWithDb,
      credentialsMode: 'flags',
      insecureTLS: false,
    });
    const insecure = buildInstallCommand({
      ...optionsWithDb,
      credentialsMode: 'flags',
      insecureTLS: true,
    });
    expect(secure).toContain('curl -fsSL ');
    expect(secure).not.toContain('curl -fsSLk');
    expect(secure).not.toContain('--pmm-server-insecure-tls');
    expect(insecure).toContain('curl -fsSLk ');
    expect(insecure).toContain('--pmm-server-insecure-tls');
  });

  test('includes DB credentials in env mode', () => {
    const cmd = buildInstallCommand({
      ...optionsWithDb,
      credentialsMode: 'env',
    });
    expect(cmd).toContain("DB_USER='pmm'");
    expect(cmd).toContain("DB_PASSWORD='secret'");
    expect(cmd).toContain("DB_HOST='127.0.0.1'");
  });

  test('uses flags mode and includes db args', () => {
    const cmd = buildInstallCommand({
      ...optionsWithDb,
      credentialsMode: 'flags',
      technology: 'postgresql',
      dbName: 'postgres',
    });
    expect(cmd).toContain('sudo -E bash -s --');
    expect(cmd).toContain('curl -fsSLk');
    expect(cmd).toContain("--pmm-server-url 'https://service_token:GLSA@pmm.example.com:443'");
    expect(cmd).toContain("--tech 'postgresql'");
    expect(cmd).toContain("--db-password 'secret'");
    expect(cmd).toContain("--db-name 'postgres'");
    expect(cmd).toContain('--pmm-server-insecure-tls');
  });

  test('includes mongodb auth db only for mongodb', () => {
    const mongodb = buildInstallCommand({
      ...optionsWithDb,
      credentialsMode: 'env',
      technology: 'mongodb',
      dbAuthDB: 'admin',
    });
    const mysql = buildInstallCommand({
      ...optionsWithDb,
      credentialsMode: 'env',
      technology: 'mysql',
      dbAuthDB: 'admin',
    });

    expect(mongodb).toContain("DB_AUTH_DB='admin'");
    expect(mysql).not.toContain('DB_AUTH_DB=');
  });

  test('supports valkey technology', () => {
    const cmd = buildInstallCommand({
      ...baseOptions,
      technology: 'valkey',
    });
    expect(cmd).toContain("TECH='valkey'");
  });

  test('includes mysql query source in env mode', () => {
    const cmd = buildInstallCommand({
      ...baseOptions,
      dbQuerySource: 'perfschema',
    });
    expect(cmd).toContain("DB_QUERY_SOURCE='perfschema'");
    expect(cmd).not.toContain('--db-query-source');
  });

  test('includes mysql query source in flags mode', () => {
    const cmd = buildInstallCommand({
      ...baseOptions,
      credentialsMode: 'flags',
      dbQuerySource: 'slowlog',
    });
    expect(cmd).toContain("--db-query-source 'slowlog'");
  });

  test('omits mysql query source for non-mysql technologies', () => {
    const cmd = buildInstallCommand({
      ...baseOptions,
      technology: 'postgresql',
      dbQuerySource: 'slowlog',
    });
    expect(cmd).not.toContain('DB_QUERY_SOURCE=');
    expect(cmd).not.toContain('--db-query-source');
  });
});

describe('buildInstallCommand prompt mode', () => {
  const promptBase: InstallCommandOptions = {
    ...baseOptions,
    credentialsMode: 'prompt',
  };

  test('renders a two-line curl-then-sudo-E-bash command', () => {
    const cmd = buildInstallCommand(promptBase);
    const lines = cmd.split('\n');
    expect(lines[0]).toContain(
      "curl -fsSLk -o '/tmp/install-pmm-client.sh' 'https://pmm.example.com/pmm-static/install-pmm-client.sh'"
    );
    expect(lines[1]).toMatch(/^sudo -E bash '\/tmp\/install-pmm-client\.sh' \\$/);
    expect(cmd).not.toContain('curl |');
    expect(cmd).not.toContain('bash -s --');
    expect(cmd).not.toContain('sudo -E env');
  });

  test('never emits DB credentials, even when fields are filled', () => {
    const cmd = buildInstallCommand({
      ...promptBase,
      dbUser: 'pmm',
      dbPassword: 'secret',
    });
    expect(cmd).not.toContain('DB_USER=');
    expect(cmd).not.toContain('DB_PASSWORD=');
    expect(cmd).not.toContain('--db-user');
    expect(cmd).not.toContain('--db-password');
    expect(cmd).not.toContain("'pmm'");
    expect(cmd).not.toContain("'secret'");
  });

  test('emits non-credential DB fields as flags', () => {
    const cmd = buildInstallCommand({
      ...promptBase,
      dbHost: '127.0.0.1',
      dbPort: '3306',
      dbServiceName: 'node-mysql',
      nodeName: 'node-1',
      nodeAddress: '10.0.0.1',
    });
    expect(cmd).toContain("--db-host '127.0.0.1'");
    expect(cmd).toContain("--db-port '3306'");
    expect(cmd).toContain("--db-service-name 'node-mysql'");
    expect(cmd).toContain("--node-name 'node-1'");
    expect(cmd).toContain("--node-address '10.0.0.1'");
  });

  test('emits --db-name only for postgresql', () => {
    const pg = buildInstallCommand({
      ...promptBase,
      technology: 'postgresql',
      dbName: 'postgres',
    });
    const mysql = buildInstallCommand({
      ...promptBase,
      technology: 'mysql',
      dbName: 'postgres',
    });
    expect(pg).toContain("--db-name 'postgres'");
    expect(mysql).not.toContain('--db-name');
  });

  test('emits --db-auth-db only for mongodb', () => {
    const mongo = buildInstallCommand({
      ...promptBase,
      technology: 'mongodb',
      dbAuthDB: 'admin',
    });
    const mysql = buildInstallCommand({
      ...promptBase,
      technology: 'mysql',
      dbAuthDB: 'admin',
    });
    expect(mongo).toContain("--db-auth-db 'admin'");
    expect(mysql).not.toContain('--db-auth-db');
  });

  test('emits --db-query-source only for mysql', () => {
    const mysql = buildInstallCommand({
      ...promptBase,
      dbQuerySource: 'perfschema',
    });
    const pg = buildInstallCommand({
      ...promptBase,
      technology: 'postgresql',
      dbQuerySource: 'perfschema',
    });
    expect(mysql).toContain("--db-query-source 'perfschema'");
    expect(pg).not.toContain('--db-query-source');
  });

  test('respects insecureTLS toggle for both curl and the script', () => {
    const secure = buildInstallCommand({ ...promptBase, insecureTLS: false });
    const insecure = buildInstallCommand({ ...promptBase, insecureTLS: true });

    expect(secure).toContain('curl -fsSL ');
    expect(secure).not.toContain('curl -fsSLk');
    expect(secure).not.toContain('--pmm-server-insecure-tls');

    expect(insecure).toContain('curl -fsSLk ');
    expect(insecure).toContain('--pmm-server-insecure-tls');
  });

  test('emits --pmm-server-url and --tech', () => {
    const cmd = buildInstallCommand(promptBase);
    expect(cmd).toContain(
      "--pmm-server-url 'https://service_token:GLSA@pmm.example.com:443'"
    );
    expect(cmd).toContain("--tech 'mysql'");
  });

  test('emits --force when registerForce is on', () => {
    const off = buildInstallCommand({ ...promptBase, registerForce: false });
    const on = buildInstallCommand({ ...promptBase, registerForce: true });
    expect(off).not.toContain('--force');
    expect(on).toContain('--force');
  });
});

describe('suggestDbServiceName', () => {
  test('uses hostname placeholder and tech label without port', () => {
    expect(suggestDbServiceName('mysql', '', '')).toBe('<hostname>-mysql');
    expect(suggestDbServiceName('postgresql', '', 'db1')).toBe('db1-postgresql');
  });

  test('appends port suffix when db port is set', () => {
    expect(suggestDbServiceName('mysql', '3307', 'db1')).toBe('db1-mysql-3307');
    expect(suggestDbServiceName('mysql', '3306', 'db1')).toBe('db1-mysql-3306');
  });

  test('ignores invalid port and falls back to base name', () => {
    expect(suggestDbServiceName('mongodb', 'abc', 'db1')).toBe('db1-mongodb');
  });

  test('default ports match install script', () => {
    expect(DEFAULT_DB_PORTS.mysql).toBe(3306);
    expect(DEFAULT_DB_PORTS.valkey).toBe(6379);
  });
});

describe('formatExpiresIn', () => {
  test('formats whole minutes with zero seconds', () => {
    expect(formatExpiresIn(15 * 60)).toBe('15:00');
    expect(formatExpiresIn(60)).toBe('1:00');
  });

  test('zero-pads seconds', () => {
    expect(formatExpiresIn(125)).toBe('2:05');
    expect(formatExpiresIn(9)).toBe('0:09');
  });

  test('handles boundary values', () => {
    expect(formatExpiresIn(0)).toBe('0:00');
    expect(formatExpiresIn(1)).toBe('0:01');
    expect(formatExpiresIn(59)).toBe('0:59');
    expect(formatExpiresIn(60)).toBe('1:00');
  });

  test('floors fractional seconds', () => {
    expect(formatExpiresIn(59.9)).toBe('0:59');
    expect(formatExpiresIn(120.4)).toBe('2:00');
  });

  test('clamps negatives to 0:00 (already expired)', () => {
    expect(formatExpiresIn(-1)).toBe('0:00');
    expect(formatExpiresIn(-9999)).toBe('0:00');
  });

  test('treats non-finite input as 0:00 (never renders NaN:NaN)', () => {
    expect(formatExpiresIn(NaN)).toBe('0:00');
    expect(formatExpiresIn(Infinity)).toBe('0:00');
  });
});

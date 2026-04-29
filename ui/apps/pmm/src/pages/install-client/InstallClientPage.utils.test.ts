import { describe, expect, test } from 'vitest';
import {
  buildInstallCommand,
  buildPmmServerURL,
  formatExpiresIn,
  InstallCommandOptions,
} from './InstallClientPage.utils';

const baseOptions: InstallCommandOptions = {
  installerUrl: 'https://pmm.example.com/pmm-static/install-pmm-client.sh',
  technology: 'mysql',
  credentialsMode: 'prompt',
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
});

describe('buildInstallCommand', () => {
  test('omits DB env in prompt mode when DB fields are empty', () => {
    const cmd = buildInstallCommand(baseOptions);
    expect(cmd).toContain("TECH='mysql'");
    expect(cmd).not.toContain('DB_PASSWORD=');
    expect(cmd).not.toContain('DB_USER=');
    expect(cmd).toContain('sudo -E env');
    expect(cmd).toContain('curl -fsSLk');
    expect(cmd).toContain('bash -s --');
    expect(cmd).toContain('--pmm-server-insecure-tls');
  });

  test('includes DB env in prompt mode when optional DB fields are set', () => {
    const cmd = buildInstallCommand(optionsWithDb);
    expect(cmd).toContain("DB_USER='pmm'");
    expect(cmd).toContain("DB_PASSWORD='secret'");
    expect(cmd).toContain("DB_HOST='127.0.0.1'");
    expect(cmd).toContain("DB_PORT='3306'");
    expect(cmd).toContain("DB_SERVICE_NAME='node-mysql'");
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
});

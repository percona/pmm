import { describe, expect, test } from 'vitest';
import {
  buildInstallCommand,
  buildPmmServerURL,
  InstallCommandOptions,
} from './InstallClientPage.utils';

const baseOptions: InstallCommandOptions = {
  installerUrl: 'https://pmm.example.com/pmm-static/install-pmm-client.sh',
  technology: 'mysql',
  credentialsMode: 'prompt',
  serverURL: 'https://service_token:GLSA@pmm.example.com:443',
  insecureTLS: false,
  registerForce: false,
  nodeName: '',
  nodeAddress: '',
  dbUser: 'pmm',
  dbPassword: 'secret',
  dbHost: '127.0.0.1',
  dbPort: '3306',
  dbName: '',
  dbAuthDB: '',
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
  test('omits DB password in prompt mode', () => {
    const cmd = buildInstallCommand(baseOptions);
    expect(cmd).toContain("TECH='mysql'");
    expect(cmd).not.toContain('DB_PASSWORD=');
    expect(cmd).toContain('sudo env');
  });

  test('includes DB credentials in env mode', () => {
    const cmd = buildInstallCommand({
      ...baseOptions,
      credentialsMode: 'env',
    });
    expect(cmd).toContain("DB_USER='pmm'");
    expect(cmd).toContain("DB_PASSWORD='secret'");
    expect(cmd).toContain("DB_HOST='127.0.0.1'");
  });

  test('uses flags mode and includes db args', () => {
    const cmd = buildInstallCommand({
      ...baseOptions,
      credentialsMode: 'flags',
      technology: 'postgresql',
      dbName: 'postgres',
    });
    expect(cmd).toContain('sudo bash -s --');
    expect(cmd).toContain("--tech 'postgresql'");
    expect(cmd).toContain("--db-password 'secret'");
    expect(cmd).toContain("--db-name 'postgres'");
  });

  test('includes mongodb auth db only for mongodb', () => {
    const mongodb = buildInstallCommand({
      ...baseOptions,
      credentialsMode: 'env',
      technology: 'mongodb',
      dbAuthDB: 'admin',
    });
    const mysql = buildInstallCommand({
      ...baseOptions,
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

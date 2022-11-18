import { test, expect } from '@playwright/test';
// import cli = require('@helpers/cliHelper'); //optional way to import with local name
import * as cli from '@helpers/cliHelper';

test.describe('Spec file for MongoDB CLI tests ', async () => {
  test('pmm-admin mongodb --help check for socket @cli @mongo', async ({}) => {
    const output = await cli.exec('pmm-admin add mongodb --help');
    /*
    echo "$output"
        [ "$status" -eq 0 ]
    [[ ${lines[0]} =~ "Usage: pmm-admin add mongodb [<name> [<address>]]" ]]
    echo "${output}" | grep -- "--socket=STRING"
*/
    await test.step('Verify "--socket=STRING" is present', async () => {
      await output.assertSuccess();
      await expect(output.stdout).toContain('Usage: pmm-admin add mongodb [<name> [<address>]]');
      await expect(output.stdout).toContain('--socket=STRING');
    });
  });

  test('run pmm-admin add mongodb --help to check metrics-mode="auto" @mongo', async ({}) => {
    const output = await cli.exec('pmm-admin add mongodb --help');
    /*
    echo "$output"
        [ "$status" -eq 0 ]
    echo "${output}" | grep "metrics-mode=\"auto\""
*/
    await test.step('Verify metrics-mode="auto" is present', async () => {
      await output.assertSuccess();
      await expect(output.stdout).toContain('Usage: pmm-admin add mongodb [<name> [<address>]]');
      await expect(output.stdout).toContain('metrics-mode="auto"');
    });
  });

  test('run pmm-admin add mongodb --help to check host @mongo', async ({}) => {
    const output = await cli.exec('pmm-admin add mongodb --help');
    /*
    echo "$output"
        [ "$status" -eq 0 ]
    echo "${output}" | grep "host"
*/
    await test.step('Verify "metrics-mode="auto"" is present', async () => {
      await output.assertSuccess();
      await expect(output.stdout).toContain('Usage: pmm-admin add mongodb [<name> [<address>]]');
      await expect(output.stdout).toContain('host');
    });
  });

  test('run pmm-admin add mongodb --help to check port @mongo', async ({}) => {
    const output = await cli.exec('pmm-admin add mongodb --help');
    /*
    echo "$output"
        [ "$status" -eq 0 ]
    echo "${output}" | grep "port"
*/
    await test.step('Verify "port" is present', async () => {
      await output.assertSuccess();
      await expect(output.stdout).toContain('Usage: pmm-admin add mongodb [<name> [<address>]]');
      await expect(output.stdout).toContain('--socket=STRING');
    });
  });

  test('run pmm-admin add mongodb --help to check service-name @mongo', async ({}) => {
    const output = await cli.exec('pmm-admin add mongodb --help');
    /*
    echo "$output"
        [ "$status" -eq 0 ]
    echo "${output}" | grep "service-name"
*/
    await test.step('Verify "service-name" is present', async () => {
      await output.assertSuccess();
      await expect(output.stdout).toContain('Usage: pmm-admin add mongodb [<name> [<address>]]');
      await expect(output.stdout).toContain('service-name');
    });
  });

  test('@PMM-T925 - Verify help for pmm-admin add mongodb has TLS-related flags @mongo', async ({}) => {
    const output = await cli.exec('pmm-admin add mongodb --help');

    /*
    echo "$output"
        [ "$status" -eq 0 ]
    echo "${output}" | grep "tls                        Use TLS to connect to the database"
    echo "${output}" | grep "tls-skip-verify            Skip TLS certificates validation"
    echo "${output}" | grep "tls-certificate-key-file=STRING"
    echo "${output}" | grep "tls-certificate-key-file-password=STRING"
    echo "${output}" | grep "tls-ca-file=STRING         Path to certificate authority file"
    echo "${output}" | grep "authentication-mechanism=STRING"
    echo "${output}" | grep "authentication-database=STRING"
*/
    await output.assertSuccess();
    await output.containsMany([
      'tls                        Use TLS to connect to the database',
      'tls-skip-verify            Skip TLS certificates validation',
      'tls-certificate-key-file=STRING',
      'tls-certificate-key-file-password=STRING',
      'tls-ca-file=STRING         Path to certificate authority file',
      'authentication-mechanism=STRING',
      'authentication-database=STRING',
    ]);
  });
});
